package stream

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// MustReadStream blocks until a non-empty line is read from stream. When
// stream is a TTY, canonical input mode is disabled for the read so the
// MAX_CANON limit (1024 bytes on macOS) doesn't truncate pasted SDPs.
func MustReadStream(stream io.Reader) (string, error) {
	if file, ok := stream.(*os.File); ok {
		if restore, ok := disableCanonicalMode(file); ok {
			defer restore()
		}
	}
	reader := bufio.NewReader(stream)

	var in string
	for {
		var err error
		in, err = reader.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				return "", err
			}
		}
		in = strings.TrimSpace(in)
		if in != "" {
			break
		}
	}

	return in, nil
}
