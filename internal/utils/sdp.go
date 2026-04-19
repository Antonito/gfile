package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pion/webrtc/v4"

	"github.com/antonito/gfile/internal/utils/base45"
)

// EncodeSDP serialises desc as JSON, zstd-compresses it,
// and base45-encodes the result.
func EncodeSDP(desc webrtc.SessionDescription) (string, error) {
	jsonBytes, err := json.Marshal(desc)
	if err != nil {
		return "", err
	}

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = enc.Close()
	}()

	compressed := enc.EncodeAll(jsonBytes, nil)

	return base45.Encode(compressed), nil
}

// ResolveSDPFlag returns the encoded SDP string referenced by value, validates
// it via DecodeSDP, and returns a descriptive error when the supplied value
// is unusable. value must be non-empty; callers are responsible for the
// "unset" branch.
//
// A leading '@' switches to file mode: @- reads stdin, @<path> reads a
// file. Otherwise the value is treated as a literal encoded SDP.
func ResolveSDPFlag(value string, stdin io.Reader) (string, error) {
	var raw string
	if strings.HasPrefix(value, "@") {
		src := value[1:]
		var (
			data []byte
			err  error
		)
		if src == "-" {
			data, err = io.ReadAll(stdin)
		} else {
			data, err = os.ReadFile(src)
		}
		if err != nil {
			return "", fmt.Errorf("read SDP source: %w", err)
		}
		raw = string(data)
	} else {
		raw = value
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("invalid SDP value: empty SDP")
	}
	if _, err := DecodeSDP(raw); err != nil {
		return "", fmt.Errorf("invalid SDP value: %w", err)
	}
	return raw, nil
}

// DecodeSDP reverses EncodeSDP.
func DecodeSDP(in string) (webrtc.SessionDescription, error) {
	var out webrtc.SessionDescription
	compressed, err := base45.Decode(in)
	if err != nil {
		return out, err
	}

	dec, err := zstd.NewReader(nil)
	if err != nil {
		return out, err
	}
	defer dec.Close()

	raw, err := dec.DecodeAll(compressed, nil)
	if err != nil {
		return out, fmt.Errorf("decompress: %w", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}

	return out, nil
}
