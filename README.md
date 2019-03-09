# GFile

gfile is a WebRTC based file exchange software.

It allows to share a file directly between two computers, without the need of a third party.

## Usage

### Sender
```bash
gfile send --file filename
```

- Run the command
- A base64 encoded SDP will appear, send it to the remote client
- Follow the instruction to curl the client's SDP to your process
- The file transfer should start

> Due to terms restrictions (ability to treat lines with +1024 characters), the SDP must be send through `curl`

### Receiver
```bash
# SDP being the base64 SDP gotten from the other client
echo "$SDP" | gfile receive -o filename
```

- Pipe the other client's SDP to gfile
- A base64 encoded SDP will appear, send it to the remote client
- The file transfer should start

# Contributors

- Antoine Baché (https://github.com/Antonito) **Original author**