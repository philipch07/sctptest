package main

import (
	"flag"

	"github.com/pion/logging"
)

var (
	network    string
	serverAddr string
	port       int
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	flag.StringVar(&network, "network", "udp4", "network type. specify `udp` for SCTP, or `tcp` for TCP")
	flag.StringVar(&serverAddr, "s", "", "Remote server address (e.g. 127.0.0.1:40916")
	flag.IntVar(&port, "l", 40916, "Port number to listen as a server")

	flag.Parse()
}

func main() {
	loggerFactory := logging.NewDefaultLoggerFactory()

	if len(serverAddr) > 0 {
		c, err := newClient(&clientConfig{
			network:       network,
			server:        serverAddr,
			loggerFactory: loggerFactory,
		})
		checkErr(err)
		err = c.start()
		checkErr(err)
	} else {
		s, err := newServer(&serverConfig{
			network:       network,
			listenPort:    port,
			loggerFactory: loggerFactory,
		})
		checkErr(err)
		err = s.start()
		checkErr(err)
	}
}
