package base45

import (
	"errors"
	"fmt"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"
	base     = 45
)

// index maps each alphabet byte to its value, or 0xff for invalid.
var index = func() [256]byte {
	var t [256]byte

	for ndx := range t {
		t[ndx] = 0xff
	}
	for ndx := range len(alphabet) {
		t[alphabet[ndx]] = byte(ndx)
	}

	return t
}()

// Errors returned by Decode.
var (
	errInvalidLength = errors.New("base45: invalid length")
	errOutOfRange    = errors.New("base45: chunk value out of range")
)

// Encode encodes src using the RFC 9285 base45 alphabet. Every two
// input bytes become three output characters; a trailing odd byte becomes
// two output characters.
func Encode(src []byte) string {
	if len(src) == 0 {
		return ""
	}
	out := make([]byte, 0, (len(src)/2)*3+(len(src)%2)*2)
	ndx := 0
	for ; ndx+1 < len(src); ndx += 2 {
		word := uint(src[ndx])<<8 | uint(src[ndx+1])
		digit0 := word % base
		digit1 := (word / base) % base
		digit2 := word / (base * base)
		out = append(out, alphabet[digit0], alphabet[digit1], alphabet[digit2])
	}
	if ndx < len(src) {
		word := uint(src[ndx])
		digit0 := word % base
		digit1 := word / base
		out = append(out, alphabet[digit0], alphabet[digit1])
	}
	return string(out)
}

// Decode reverses Encode. Input length mod 3 must be 0 or 2.
func Decode(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	rem := len(encoded) % 3
	if rem == 1 {
		return nil, errInvalidLength
	}

	out := make([]byte, 0, (len(encoded)/3)*2+(rem/2))
	ndx := 0
	for ; ndx+3 <= len(encoded); ndx += 3 {
		word, err := decodeChunk(encoded[ndx : ndx+3])
		if err != nil {
			return nil, err
		}
		if word > 0xffff {
			return nil, errOutOfRange
		}
		out = append(out, byte(word>>8), byte(word))
	}
	if rem == 2 {
		word, err := decodeChunk(encoded[ndx : ndx+2])
		if err != nil {
			return nil, err
		}
		if word > 0xff {
			return nil, errOutOfRange
		}
		out = append(out, byte(word))
	}
	return out, nil
}

// decodeChunk interprets chunk as base-45 digits, least-significant first,
// and returns the accumulated value.
func decodeChunk(chunk string) (uint, error) {
	var acc uint
	for ndx := len(chunk) - 1; ndx >= 0; ndx-- {
		digit := index[chunk[ndx]]
		if digit == 0xff {
			return 0, errInvalidChar(chunk[ndx])
		}
		acc = acc*base + uint(digit)
	}
	return acc, nil
}

func errInvalidChar(char byte) error {
	return fmt.Errorf("base45: invalid character %q", char)
}
