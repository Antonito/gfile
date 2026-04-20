[![CI](https://github.com/Antonito/gfile/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/Antonito/gfile/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Antonito/gfile)](https://goreportcard.com/report/github.com/Antonito/gfile)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)  

# GFile

gfile is a WebRTC-based file exchange tool.

It lets you share a file directly between two computers, without the need of a third party.

![ezgif-5-9936f8008e4d](https://user-images.githubusercontent.com/11705040/55694519-686e2d80-5969-11e9-9bc1-f7a59b62732f.gif)

## Note

This project is still in its early stage.

## How does it work ?

![Schema](https://user-images.githubusercontent.com/11705040/55741923-4dd89a80-59e3-11e9-917c-daf9f08f164d.png)

The [STUN server](https://en.wikipedia.org/wiki/STUN) is only used to help the two clients find each other through NAT. The data you transfer with `gfile` **does not transit through it**.

> More information [here](https://webrtc.org/)

For wire-format and session-protocol details, see [PROTOCOL.md](PROTOCOL.md).

### STUN configuration

`gfile` defaults to Google's public STUN server (`stun.l.google.com:19302`). You can override this with `--stun`:

```bash
# Use a specific STUN server
gfile --stun stun.cloudflare.com:3478 send -f filename

# Use multiple STUN servers (comma-separated)
gfile --stun stun.l.google.com:19302,stun.cloudflare.com:3478 send -f filename

# Disable STUN entirely — host/mDNS candidates only (LAN use)
gfile --stun="" send -f filename
```

### mDNS

By default `gfile` advertises host ICE candidates as `.local` mDNS hostnames instead of raw LAN IPs, matching modern browser behavior. This keeps your internal network topology out of the SDP you paste around, and is enough for a LAN transfer even with `--stun=""`.

Disable it if a peer on the other side cannot resolve mDNS:

```bash
gfile --mdns=false send -f filename
```

## Usage

### Sender

```bash
gfile send --file filename
```

-   Run the command
-   A compact encoded [SDP](https://tools.ietf.org/html/rfc4566) will appear, send it to the remote client
-   Follow the instructions to send the client's SDP to your process
-   The file transfer should start

Pass `--connections N` (1..16) to open `N` parallel peer connections. The default is 1. Higher values can improve throughput on high-latency or high-bandwidth links — see [PROTOCOL.md](PROTOCOL.md#multi-pc-mode).

### Receiver

```bash
# SDP being the compact encoded SDP gotten from the other client
echo "$SDP" | gfile receive -o filename
```

-   Pipe the other client's SDP to gfile
-   A compact encoded SDP will appear, send it to the remote client
-   The file transfer should start

### Benchmark

`gfile` can benchmark the network speed between two clients (one _sender_, one _receiver_) with the `bench` command. The SDP exchange works the same as in `send` / `receive`.

This feature is still an experiment.

```bash
# Run as 'sender'
gfile bench -s

# Run as 'receiver'
echo "$SDP" |  gfile bench
```

### Debug

For more verbose output, set the logging level via the `GFILE_LOG` environment variable.

> Example: `export GFILE_LOG="TRACE"`
> See function `setupLogger` in  `main.go` for more information

## Contributors

-   Antoine Baché ([https://github.com/Antonito](https://github.com/Antonito)) **Original author**

Special thanks to [Sean DuBois](https://github.com/Sean-Der) for his help with [pion/webrtc](https://github.com/pion/webrtc) and [Yutaka Takeda](https://github.com/enobufs) for his work on [pion/sctp](https://github.com/pion/sctp)
