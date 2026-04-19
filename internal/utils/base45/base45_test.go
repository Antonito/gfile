package base45_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonito/gfile/internal/utils/base45"
)

// RFC 9285 §4.3 encoding examples.
var vectors = []struct {
	plain   string
	encoded string
}{
	{"AB", "BB8"},
	{"Hello!!", "%69 VD92EX0"},
	{"base-45", "UJCLQE7W581"},
	{"ietf!", "QED8WEX0"},
}

func Test_Encode(t *testing.T) {
	for _, tc := range vectors {
		got := base45.Encode([]byte(tc.plain))
		assert.Equal(t, tc.encoded, got, "encode %q", tc.plain)
	}
}

func Test_Encode_Empty(t *testing.T) {
	assert.Empty(t, base45.Encode(nil))
	assert.Empty(t, base45.Encode([]byte{}))
}

func Test_Encode_Lengths(t *testing.T) {
	tests := []struct {
		in      []byte
		wantLen int
	}{
		// leftover byte → 2 chars
		{[]byte{0x00}, 2},
		// leftover byte → 2 chars
		{[]byte{0xFF}, 2},
		// one pair → 3 chars
		{[]byte{0x00, 0x00}, 3},
		// pair + leftover → 5 chars
		{[]byte{0x00, 0x00, 0x00}, 5},
	}
	for _, tc := range tests {
		assert.Len(t, base45.Encode(tc.in), tc.wantLen)
	}
}

func Test_Decode(t *testing.T) {
	for _, tc := range vectors {
		got, err := base45.Decode(tc.encoded)
		require.NoError(t, err, "decode %q", tc.encoded)
		assert.Equal(t, []byte(tc.plain), got, "decode %q", tc.encoded)
	}
}

func Test_Decode_Empty(t *testing.T) {
	got, err := base45.Decode("")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func Test_RoundTrip(t *testing.T) {
	// Byte-wise exhaustive: encode/decode every single-byte input.
	for byteVal := range 0x100 {
		enc := base45.Encode([]byte{byte(byteVal)})
		back, err := base45.Decode(enc)
		require.NoError(t, err, "byte=%d", byteVal)
		assert.Equal(t, []byte{byte(byteVal)}, back, "byte=%d", byteVal)
	}
	// And every two-byte pair, sampled.
	for byteA := 0; byteA <= 0xFF; byteA += 17 {
		for byteB := 0; byteB <= 0xFF; byteB += 13 {
			in := []byte{byte(byteA), byte(byteB)}
			back, err := base45.Decode(base45.Encode(in))
			require.NoError(t, err)
			assert.Equal(t, in, back)
		}
	}
}

func Test_Decode_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"lowercase letter", "bb8"},
		{"base64 padding", "BB8="},
		{"base64 slash", "BB/"},
		{"base64 plus inside garbage", "AA+AA"},
		{"single char", "A"},
		{"four chars", "BB8A"},
		// chunk value overflow: "ZZZ" = 44 + 44*45 + 44*45*45 = 89144 > 65535
		{"three-char overflow", "ZZZ"},
		// two-char overflow: "ZZ" = 44 + 44*45 = 2024, but only 0..255 allowed
		{"two-char overflow", "ZZ"},
	}
	for _, tc := range tests {
		_, err := base45.Decode(tc.input)
		assert.Error(t, err, tc.name)
	}
}
