package main

import (
	"log"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/enobufs/go-rudp"
	"github.com/pion/logging"
)

type clientConfig struct {
	network       string
	server        string
	loggerFactory logging.LoggerFactory
}

type client struct {
	network       string
	remAddr       *net.UDPAddr
	log           logging.LeveledLogger
	loggerFactory logging.LoggerFactory
}

func newClient(cfg *clientConfig) (*client, error) {
	remAddr, err := net.ResolveUDPAddr(cfg.network, cfg.server)
	if err != nil {
		return nil, err
	}

	return &client{
		network:       cfg.network,
		remAddr:       remAddr,
		log:           cfg.loggerFactory.NewLogger("client"),
		loggerFactory: cfg.loggerFactory,
	}, nil
}

func (c *client) start() error {
	log.Printf("connecting to server %s ...", c.remAddr.String())
	rudpc, err := rudp.Dial(&rudp.DialConfig{
		Network:       c.network,
		RemoteAddr:    c.remAddr,
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
	msgSize := 32 * 1024
	totalNumMsgs := 64 * 1024 // 2GB
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

	buf := make([]byte, msgSize)
	for i := 0; i < totalNumMsgs; i++ {
		_, err := rnd.Read(buf)
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

func throughputTicker(totalBytes *uint64) *time.Ticker {
	since := time.Now()
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case _, ok := <-ticker.C:
				if !ok {
					return
				}
			}
			bps := float64(atomic.LoadUint64(totalBytes)*8) / time.Since(since).Seconds()
			log.Printf("Throughput: %.03f Mbps", bps/1024/1024)
		}
	}()
	return ticker
}
