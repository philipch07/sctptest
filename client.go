package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/enobufs/go-rudp"
	"github.com/pion/logging"
)

type clientConfig struct {
	network       string
	server        string
	bufferSize    int
	loggerFactory logging.LoggerFactory
}

type client interface {
	start(time.Duration) error
}

type sctpClient struct {
	network       string
	remAddr       *net.UDPAddr
	bufferSize    int
	log           logging.LeveledLogger
	loggerFactory logging.LoggerFactory
}

type tcpClient struct {
	network string
	remAddr *net.TCPAddr
	log     logging.LeveledLogger
}

func newClient(cfg *clientConfig) (client, error) {
	if strings.HasPrefix(cfg.network, "udp") {
		remAddr, err := net.ResolveUDPAddr(cfg.network, cfg.server)
		if err != nil {
			return nil, err
		}
		return &sctpClient{
			network:       cfg.network,
			remAddr:       remAddr,
			bufferSize:    cfg.bufferSize,
			log:           cfg.loggerFactory.NewLogger("client"),
			loggerFactory: cfg.loggerFactory,
		}, nil
	}

	if strings.HasPrefix(cfg.network, "tcp") {
		remAddr, err := net.ResolveTCPAddr(cfg.network, cfg.server)
		if err != nil {
			return nil, err
		}
		return &tcpClient{
			network: cfg.network,
			remAddr: remAddr,
			log:     cfg.loggerFactory.NewLogger("client"),
		}, nil
	}

	return nil, fmt.Errorf("invalid network %s", cfg.network)
}

func (c *sctpClient) start(duration time.Duration) error {
	log.Printf("connecting to server %s ...", c.remAddr.String())
	rudpc, err := rudp.Dial(&rudp.DialConfig{
		Network:       c.network,
		RemoteAddr:    c.remAddr,
		BufferSize:    c.bufferSize,
		LoggerFactory: c.loggerFactory,
	})
	if err != nil {
		return err
	}
	defer rudpc.Close()

	clientCh, err := rudpc.OpenChannel(777)
	if err != nil {
		return err
	}

	maxBufferAmount := uint64(1024 * 1024)
	bufferedAmountTh := uint64(512 * 1024)
	var totalBytesSent uint64
	writable := make(chan struct{}, 1)

	clientCh.SetBufferedAmountLowThreshold(bufferedAmountTh)
	clientCh.OnBufferedAmountLow(func() {
		select {
		case writable <- struct{}{}:
		default:
		}
	})

	// Start printing out the observed throughput
	ticker := throughputTicker(&totalBytesSent)

	src := rand.NewSource(123)
	rnd := rand.New(src)

	since := time.Now()

	buf := make([]byte, msgSize)
	for i := 0; i < numMsgs && (duration == 0 || time.Since(since) < duration); i++ {
		_, err := rnd.Read(buf)
		if err != nil {
			panic(err)
		}
		if clientCh.BufferedAmount() >= maxBufferAmount {
			<-writable
		}
		n, err := clientCh.Write(buf)
		if err != nil {
			panic(err)
		}
		atomic.AddUint64(&totalBytesSent, uint64(n))
	}

	ticker.Stop()

	// Wait until the buffer is completely drained
	for clientCh.BufferedAmount() > 0 {
		time.Sleep(time.Second)
	}

	log.Println("client done")
	close(writable)
	clientCh.Close()
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (c *tcpClient) start(duration time.Duration) error {
	log.Printf("connecting to server %s ...", c.remAddr.String())
	locAddr := &net.TCPAddr{
		Port: 0,
	}
	tcpc, err := net.DialTCP(c.network, locAddr, c.remAddr)
	if err != nil {
		return err
	}
	defer tcpc.Close()

	var totalBytesSent uint64

	// Start printing out the observed throughput
	ticker := throughputTicker(&totalBytesSent)

	src := rand.NewSource(123)
	rnd := rand.New(src)

	since := time.Now()

	buf := make([]byte, msgSize)
	for i := 0; i < numMsgs && (duration == 0 || time.Since(since) < duration); i++ {
		_, err := rnd.Read(buf)
		if err != nil {
			panic(err)
		}
		var nWritten int
		for nWritten < msgSize {
			n, err := tcpc.Write(buf[nWritten:])
			if err != nil {
				panic(err)
			}
			atomic.AddUint64(&totalBytesSent, uint64(n))
			nWritten += n
		}
	}

	ticker.Stop()

	// Shutdown gracefully
	tcpc.CloseWrite()
	for {
		_, err := tcpc.Read(buf)
		if err != nil {
			break
		}
	}

	log.Println("client done")
	tcpc.CloseRead()
	return nil
}

func throughputTicker(totalBytes *uint64) *time.Ticker {
	ticker := time.NewTicker(2 * time.Second)
	lastBytes := atomic.LoadUint64(totalBytes)
	go func() {
		for {
			since := time.Now()
			select {
			case _, ok := <-ticker.C:
				if !ok {
					return
				}
			}
			totalBytes := atomic.LoadUint64(totalBytes)
			bps := float64((totalBytes-lastBytes)*8) / time.Since(since).Seconds()
			lastBytes = totalBytes
			log.Printf("Throughput: %.03f Mbps", bps/1024/1024)
		}
	}()
	return ticker
}
