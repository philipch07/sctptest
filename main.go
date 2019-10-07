package main

import (
	"flag"
	"time"

	"github.com/pion/logging"
)

var (
	network    string // used by both client and server
	serverAddr string // used by server
	port       int    // used by server
	msgSize    int    // used by client
	numMsgs    int    // used by client
	bufferSize int    // used for UDPConn
	shutDownIn int    // in seconds (0: default to no shutdown)
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	flag.StringVar(&network, "network", "udp4", "network type. specify `udp` for SCTP, or `tcp` for TCP")
	flag.StringVar(&serverAddr, "s", "", "Remote server address (e.g. 127.0.0.1:40916)")
	flag.IntVar(&port, "l", 40916, "Port number to listen as a server")
	flag.IntVar(&msgSize, "m", 32768, "Message size")
	flag.IntVar(&numMsgs, "n", 32768, "Number of messages to transfer")
	flag.IntVar(&bufferSize, "b", 0, "UDP read/write buffer size (0: use default)")
	flag.IntVar(&shutDownIn, "k", 0, "Shutdown timer in seconds (0: defaults to no-shutdown)")

	flag.Parse()
}

func main() {
	loggerFactory := logging.NewDefaultLoggerFactory()

	if len(serverAddr) > 0 {
		c, err := newClient(&clientConfig{
			network:       network,
			server:        serverAddr,
			bufferSize:    bufferSize,
			loggerFactory: loggerFactory,
		})
		checkErr(err)
		err = c.start(time.Duration(shutDownIn) * time.Second)
		checkErr(err)
	} else {
		s, err := newServer(&serverConfig{
			network:       network,
			listenPort:    port,
			bufferSize:    bufferSize,
			loggerFactory: loggerFactory,
		})
		checkErr(err)
		err = s.start(time.Duration(shutDownIn) * time.Second)
		checkErr(err)
	}
}
