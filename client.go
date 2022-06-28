package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/enobufs/go-rudp"
	"github.com/pion/logging"
)

type clientConfig struct {
	network       string
	server        string
	bufferSize    int
	chConfig      *rudp.Config
	loggerFactory logging.LoggerFactory
}

type client interface {
	start(time.Duration) error
}

type sctpClient struct {
	network       string
	remAddr       *net.UDPAddr
	bufferSize    int
	chConfig      *rudp.Config
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
			chConfig:      cfg.chConfig,
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
	defer func() {
		rudpc.Close()
	}()

	var chConfig rudp.Config
	if c.chConfig != nil {
		chConfig = *c.chConfig
	}

	clientCh, err := rudpc.OpenChannel(777, chConfig)
	if err != nil {
		return err
	}

	var bufferedAmountTh uint64 = 1024 * 1024
	var maxBufferAmount uint64 = 2 * bufferedAmountTh

	if c.bufferSize > 0 {
		maxBufferAmount = uint64(c.bufferSize * 2)
		bufferedAmountTh = uint64(c.bufferSize)
	}

	var totalBytesSent uint64
	writable := make(chan struct{}, 1)

	var done uint32
	clientCh.SetBufferedAmountLowThreshold(bufferedAmountTh)
	clientCh.OnBufferedAmountLow(func() {
		if atomic.LoadUint32(&done) != 0 {
			clientCh.OnBufferedAmountLow(nil)
			close(writable)
			return
		}
		select {
		case writable <- struct{}{}:
		default:
		}
	})

	var wg sync.WaitGroup

	// read loop
	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := make([]byte, 64*1024)
		for {
			_, err2 := clientCh.Read(buf)
			if err2 != nil {
				break
			}
		}
	}()

	// write loop
	wg.Add(1)
	go func() {
		defer wg.Done()

		src := rand.NewSource(123)
		rnd := rand.New(src)

		buf := make([]byte, msgSize)
		for atomic.LoadUint32(&done) == 0 {
			_, err2 := rnd.Read(buf)
			if err2 != nil {
				panic(err2)
			}
			if clientCh.BufferedAmount() >= maxBufferAmount {
				<-writable
			}
			n, err2 := clientCh.Write(buf)
			if err2 != nil {
				break
			}
			atomic.AddUint64(&totalBytesSent, uint64(n))
		}
	}()

	<-time.NewTimer(duration).C
	atomic.AddUint32(&done, 1)
	clientCh.Close()
	wg.Wait()
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

	var wg sync.WaitGroup
	timer := time.NewTimer(duration)

	src := rand.NewSource(123)
	rnd := rand.New(src)

	// read loop
	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := make([]byte, 64*1024)
		for {
			_, err2 := tcpc.Read(buf)
			if err2 != nil {
				break
			}
		}
	}()

	// write loop
	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := make([]byte, msgSize)
		for {
			_, err := rnd.Read(buf)
			if err != nil {
				panic(err)
			}
			var nWritten int
			for nWritten < msgSize {
				n, err := tcpc.Write(buf[nWritten:])
				if err != nil {
					return
				}
				atomic.AddUint64(&totalBytesSent, uint64(n))
				nWritten += n
			}
		}
	}()

	<-timer.C

	// Shutdown gracefully
	tcpc.CloseWrite()
	wg.Wait()
	tcpc.CloseRead()
	return nil
}
