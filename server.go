package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync/atomic"

	"github.com/enobufs/go-rudp"
	"github.com/pion/logging"
)

type serverConfig struct {
	network       string
	listenPort    int
	loggerFactory logging.LoggerFactory
}

type server struct {
	network       string
	listenPort    int
	loggerFactory logging.LoggerFactory
	log           logging.LeveledLogger
}

func newServer(cfg *serverConfig) (*server, error) {
	return &server{
		network:       cfg.network,
		listenPort:    cfg.listenPort,
		loggerFactory: cfg.loggerFactory,
		log:           cfg.loggerFactory.NewLogger("server"),
	}, nil
}

func (s *server) start() error {
	locAddr, err := net.ResolveUDPAddr(s.network, fmt.Sprintf(":%d", s.listenPort))
	if err != nil {
		return err
	}

	// make sure the server is already listening when this function returns
	l, err := rudp.Listen(&rudp.ListenConfig{
		Network:       s.network,
		LocalAddr:     locAddr,
		LoggerFactory: s.loggerFactory,
	})
	if err != nil {
		return err
	}
	defer l.Close()

	log.Printf("listening on %s ...", l.LocalAddr().String())

	for {
		sconn, err := l.Accept()
		if err != nil {
			break
		}

		log.Printf("new connecton from %s ...", sconn.RemoteAddr().String())

		go func() {
			defer sconn.Close()

			// assume 1 channel per client
			serverCh, err := sconn.AcceptChannel()
			if err != nil {
				return
			}
			defer serverCh.Close()

			var totalBytesReceived uint64
			src := rand.NewSource(123)
			rnd := rand.New(src)
			exp := make([]byte, 64*1024)

			// Start printing out the observed throughput
			ticker := throughputTicker(&totalBytesReceived)

			buf := make([]byte, 64*1024)
			for {
				n, err := serverCh.Read(buf)
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
