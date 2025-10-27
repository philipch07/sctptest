# Forked from https://github.com/enobufs/sctptest

# sctptest

User-space SCTP (pion/sctp wrapped by go-rudp) test tool.

## Features
* Output send / receive throughput to stdout every 2 seconds
* Select TCP for comparison
* Select IPv4 and IPv6
* Server/Client 1:N connections
* Change buffer size of SCTP and UDP socket

## Build
Clone the repo, cd into it, then run `go build`.

## Usage
```
$ ./sctptest -h
Usage of ./sctptest:
  -network <string>
      Network type. specify "udp" for SCTP, or "tcp" for TCP (default "udp4")
  -s <string>
      Remote server address (e.g. 127.0.0.1:40916)
  -l <int>
      Port number to listen as a server (default 40916)
  -m <int>
      Message size (default 32768)
  -b <int>
      UDP read/write buffer size (0: use default)
  -t <int>
      Duration timer in seconds (default: 30 seconds)
  -u
      Unordered (false: ordered (default), true: unordered)
  -maxRetransmits
      Partial reliability maxRetransmits (set to false by default)
  -maxPacketLifetime
      Partial reliability maxPacketLifetime (set to false by default)
```

## Run
Server terminal (the receiver):
```sh
$ ./sctptest -network udp4
2025/10/27 18:38:07 listening on udp:0.0.0.0:40916 ...
```

Client terminal (the sender):
```sh
$ ./sctptest -network udp4 -s 127.0.0.1:40916 -m 1200
2025/10/27 18:38:10 connecting to server 127.0.0.1:40916 ...
```

Note that sometimes the client closes properly after running and sometimes it doesn't, but it doesn't bother me that much. If someone wants to fix it, please feel free to make a pr for it.

Then view the throughput on the server:
```sh
2025/10/27 18:38:10 new connecton from 127.0.0.1:57040 ...
2025/10/27 18:38:10 new channel: unordered=false Reliable (556e6b6e6f776e)
2025/10/27 18:38:11 Throughput: 936.793 Mbps
2025/10/27 18:38:12 Throughput: 989.963 Mbps
2025/10/27 18:38:13 Throughput: 995.149 Mbps
...
2025/10/27 18:38:35 Throughput: 954.671 Mbps
2025/10/27 18:38:36 Throughput: 1025.224 Mbps
2025/10/27 18:38:37 Throughput: 946.623 Mbps
2025/10/27 18:38:38 closed connecton for 127.0.0.1:57040
```
