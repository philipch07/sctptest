package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/philipch07/go-rudp"
	"github.com/pion/logging"
)

const defaultMaxSCTPMessageSize = 1073741823

type serverConfig struct {
	network       string
	listenPort    int
	bufferSize    int
	loggerFactory logging.LoggerFactory
}

type server interface {
	start(time.Duration) error
}

type sctpServer struct {
	network       string
	listenPort    int
	bufferSize    int
	log           logging.LeveledLogger
	loggerFactory logging.LoggerFactory
}

type tcpServer struct {
	network    string
	listenPort int
	log        logging.LeveledLogger
}

func newServer(cfg *serverConfig) (server, error) {
	if strings.HasPrefix(cfg.network, "udp") {
		return &sctpServer{
			network:       cfg.network,
			listenPort:    cfg.listenPort,
			bufferSize:    cfg.bufferSize,
			log:           cfg.loggerFactory.NewLogger("server"),
			loggerFactory: cfg.loggerFactory,
		}, nil
	}

	if strings.HasPrefix(cfg.network, "tcp") {
		return &tcpServer{
			network:    cfg.network,
			listenPort: cfg.listenPort,
			log:        cfg.loggerFactory.NewLogger("server"),
		}, nil
	}

	return nil, fmt.Errorf("invalid network %s", cfg.network)
}

func (s *sctpServer) start(duration time.Duration) error {
	locAddr, err := net.ResolveUDPAddr(s.network, fmt.Sprintf(":%d", s.listenPort))
	if err != nil {
		return err
	}

	// make sure the server is already listening when this function returns
	l, err := rudp.Listen(&rudp.ListenConfig{
		Network:       s.network,
		BufferSize:    s.bufferSize,
		LocalAddr:     locAddr,
		LoggerFactory: s.loggerFactory,
	})
	if err != nil {
		return err
	}
	defer l.Close()

	log.Printf("listening on %s:%s ...", l.LocalAddr().Network(), l.LocalAddr().String())

	for {
		sconn, err := l.Accept()
		if err != nil {
			break
		}

		log.Printf("new connecton from %s ...", sconn.RemoteAddr().String())

		since := time.Now()

		go func(sconn *rudp.Server) {
			defer sconn.Close()

			// assume 1 channel per client
			serverCh, err := sconn.AcceptChannel()
			if err != nil {
				return
			}
			defer serverCh.Close()

			chConfig := serverCh.Config()
			unordered := (chConfig.ChannelType&0x80 != 0)
			prName := "Reliable"
			if chConfig.ChannelType&0x01 != 0 {
				prName = fmt.Sprintf("maxRetransmits=%d", chConfig.ReliabilityParameter)
			} else if chConfig.ChannelType&0x02 != 0 {
				prName = fmt.Sprintf("maxPacketLifeTime=%d", chConfig.ReliabilityParameter)
			}

			log.Printf("new channel: unordered=%v %s (%02x)", unordered, prName, chConfig.ChannelType)

			var totalBytesReceived uint64
			src := rand.NewSource(123)
			rnd := rand.New(src)
			exp := make([]byte, defaultMaxSCTPMessageSize)

			// Start printing out the observed throughput
			ticker := throughputTicker(&totalBytesReceived)

			buf := make([]byte, defaultMaxSCTPMessageSize)
			for duration == 0 || time.Since(since) < duration {
				n, err := serverCh.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("read err: %v", err)
					}
					break
				}

				if chConfig.ChannelType == 0 {
					_, err = rnd.Read(exp[:n])
					if err != nil {
						break
					}

					if !bytes.Equal(buf[:n], exp[:n]) {
						panic(fmt.Errorf("data mismatch"))
					}
				}

				atomic.AddUint64(&totalBytesReceived, uint64(n))
			}

			ticker.Stop()
			log.Printf("closed connection for %s", sconn.RemoteAddr().String())
		}(sconn)
	}

	return nil
}

func (s *tcpServer) start(duration time.Duration) error {
	locAddr, err := net.ResolveTCPAddr(s.network, fmt.Sprintf(":%d", s.listenPort))
	if err != nil {
		return err
	}

	// make sure the server is already listening when this function returns
	l, err := net.ListenTCP(s.network, locAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	log.Printf("listening on %s:%s ...", l.Addr().Network(), l.Addr().String())

	for {
		sconn, err := l.Accept()
		if err != nil {
			break
		}

		log.Printf("new connection from %s ...", sconn.RemoteAddr().String())

		since := time.Now()

		go func() {
			defer sconn.Close()

			var totalBytesReceived uint64
			src := rand.NewSource(123)
			rnd := rand.New(src)
			exp := make([]byte, defaultMaxSCTPMessageSize)

			// Start printing out the observed throughput
			ticker := throughputTicker(&totalBytesReceived)

			buf := make([]byte, defaultMaxSCTPMessageSize)
			for duration == 0 || time.Since(since) < duration {
				n, err := sconn.Read(buf)
				if err != nil {
					break
				}
				_, err = rnd.Read(exp[:n])
				if err != nil {
					break
				}
				if !bytes.Equal(buf[:n], exp[:n]) {
					panic(fmt.Errorf("data mismatch"))
				}

				atomic.AddUint64(&totalBytesReceived, uint64(n))
			}

			ticker.Stop()
			log.Printf("closed connecton for %s", sconn.RemoteAddr().String())
		}()
	}

	return nil
}

func throughputTicker(totalBytes *uint64) *time.Ticker {
	ticker := time.NewTicker(1 * time.Second)
	lastBytes := atomic.LoadUint64(totalBytes)
	prev := time.Now()

	go func() {
		for range ticker.C {
			now := time.Now()
			cur := atomic.LoadUint64(totalBytes)
			secs := now.Sub(prev).Seconds()
			if secs <= 0 {
				continue
			}
			bps := float64((cur-lastBytes)*8) / secs
			lastBytes = cur
			prev = now
			log.Printf("Throughput: %.03f Mbps", bps/1024/1024)
		}
	}()

	return ticker
}
