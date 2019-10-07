# sctptest

User-sapce SCTP (pion/sctp wrapped by go-rudp) test tool.

## Features
* Output send / receive throughput to stdout every 2 seconds
* Select TCP for comparison
* Select IPv4 and IPv6
* Server/Client 1:N connections
* Change buffer size of SCTP and UDP socket

## Build
```sh
$ git clone git@github.com:enobufs/sctptest.git
$ cd sctptest
$ go build
```

## Usage
```
$ ./sctptest -h
Usage of ./sctptest:
  -b int
    	UDP read/write buffer size (0: use default)
  -k int
    	Shutdown timer in seconds (0: defaults to no-shutdown)
  -l int
    	Port number to listen as a server (default 40916)
  -m int
    	Message size (default 32768)
  -n int
    	Number of messages to transfer (default 32768)
  -network string
    	Network type. specify "udp" for SCTP, or "tcp" for TCP (default "udp4")
  -s string
    	Remote server address (e.g. 127.0.0.1:40916)
```

## Run
Server terminal (the receiver):
```sh
$ ./sctptest -network udp4
2019/09/30 22:13:24 listening on udp:0.0.0.0:40916 ...
```

Client terminal (the sender):
```sh
$ ./sctptest -network udp4 -s 127.0.0.1:40916 -m 1200 -n 1000000
2019/09/30 22:13:34 connecting to server :40916 ...
2019/09/30 22:13:35 Throughput: 439.222 Mbps
2019/09/30 22:13:36 Throughput: 511.782 Mbps
2019/09/30 22:13:37 Throughput: 512.076 Mbps
  :
```

