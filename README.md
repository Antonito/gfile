[![Go Report Card](https://goreportcard.com/badge/github.com/Antonito/gfile)](https://goreportcard.com/report/github.com/Antonito/gfile)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/5888662aebd54d2681f9a737dfd33913)](https://www.codacy.com/app/Antonito/gfile?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Antonito/gfile&amp;utm_campaign=Badge_Grade)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# GFile

gfile is a WebRTC based file exchange software.

It allows to share a file directly between two computers, without the need of a third party.

## Note

This project is still in its early stage.

As of today, it works well with small files. It doesn't work with huge file, due to disconnection issues. (WIP)

## Usage

### Sender

```bash
gfile send --file filename
```

-   Run the command
-   A base64 encoded SDP will appear, send it to the remote client
-   Follow the instruction to curl the client's SDP to your process
-   The file transfer should start

> Due to terms restrictions (ability to treat lines with +1024 characters), the SDP must be send through `curl`

### Receiver

```bash
# SDP being the base64 SDP gotten from the other client
echo "$SDP" | gfile receive -o filename
```

-   Pipe the other client's SDP to gfile
-   A base64 encoded SDP will appear, send it to the remote client
-   The file transfer should start

## Contributors

-   Antoine Baché ([https://github.com/Antonito](https://github.com/Antonito)) **Original author**