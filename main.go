package main

import (
	"flag"
	"time"

	"github.com/enobufs/go-rudp"
	dcep "github.com/pion/datachannel"
	"github.com/pion/logging"
)

var (
	network           string // used by both client and server
	serverAddr        string // used by server
	port              int    // used by server
	msgSize           int    // used by client
	bufferSize        int    // used for UDPConn
	duration          int    // in seconds (default: 30 seconds)
	unordered         bool
	maxRetransmits    int64 // Partial reliability rexmit. Defaults to -1 (disabled)
	maxPacketLifeTime int64 // Partial reliability maxPacketLifeTime. Defaults to -1 (disabled)
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	flag.StringVar(&network, "network", "udp4", "Network type. specify \"udp\" for SCTP, or \"tcp\" for TCP")
	flag.StringVar(&serverAddr, "s", "", "Remote server address (e.g. 127.0.0.1:40916)")
	flag.IntVar(&port, "l", 40916, "Port number to listen as a server")
	flag.IntVar(&msgSize, "m", 32768, "Message size")
	flag.IntVar(&bufferSize, "b", 0, "SCTP read/write buffer size (0: use default)")
	flag.IntVar(&duration, "t", 30, "Duration timer in seconds (default: 30 seconds)")
	flag.BoolVar(&unordered, "unordered", false, "Unordered (false: ordered (default), true: unordered)")
	flag.Int64Var(&maxRetransmits, "maxRetransmits", int64(-1), "Partial reliability maxRetransmits (set to false by default)")
	flag.Int64Var(&maxPacketLifeTime, "maxPacketLifeTime", int64(-1), "Partial reliability maxPacketLifeTime (set to false by default)")

	flag.Parse()
}

func main() {
	loggerFactory := logging.NewDefaultLoggerFactory()

	var chType dcep.ChannelType
	var relParam uint32
	if unordered {
		chType |= 0x80
	}
	if maxRetransmits >= 0 {
		chType |= 0x01
		relParam = uint32(maxRetransmits)
	} else if maxPacketLifeTime >= 0 {
		chType |= 0x02
		relParam = uint32(maxPacketLifeTime)
	}

	chConfig := &rudp.Config{
		ChannelType:          chType,
		ReliabilityParameter: relParam,
	}

	if len(serverAddr) > 0 {
		c, err := newClient(&clientConfig{
			network:       network,
			server:        serverAddr,
			bufferSize:    bufferSize,
			chConfig:      chConfig,
			loggerFactory: loggerFactory,
		})
		checkErr(err)
		err = c.start(time.Duration(duration) * time.Second)
		checkErr(err)
	} else {
		s, err := newServer(&serverConfig{
			network:       network,
			listenPort:    port,
			bufferSize:    bufferSize,
			loggerFactory: loggerFactory,
		})
		checkErr(err)
		err = s.start(time.Duration(duration) * time.Second)
		checkErr(err)
	}
}
