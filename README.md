# sctptest

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
  -l int
    	Port number to listen as a server (default 40916)
  -network string
    	network type either "udp", "udp4" or "udp6" (default "udp4")
  -s string
    	Remote server address (e.g. 127.0.0.1:40916
```

## Run
Server terminal (the receiver):
```sh
$ ./sctptest
2019/09/30 22:13:24 listening on 0.0.0.0:40916 ...
```

Client terminal (the sender):
```sh
$ ./sctptest -s 127.0.0.1:40916
2019/09/30 22:13:34 connecting to server :40916 ...
2019/09/30 22:13:35 Throughput: 439.222 Mbps
2019/09/30 22:13:36 Throughput: 511.782 Mbps
2019/09/30 22:13:37 Throughput: 512.076 Mbps
  :
```

