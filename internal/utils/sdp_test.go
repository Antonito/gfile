package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_EncodeDecodeSDP(t *testing.T) {
	asrt := assert.New(t)

	input := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  "v=0\r\no=- 1 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n",
	}

	encoded, err := EncodeSDP(input)
	require.NoError(t, err)
	asrt.NotEmpty(encoded)

	// Sanity: base45 alphabet has no lowercase letters and no "=".
	// Any of these would force QR back into byte mode and undo this change.
	// Note: "/" is part of the RFC 9285 / QR alphanumeric-mode alphabet.
	asrt.Equal(strings.ToUpper(encoded), encoded, "base45 output must be uppercase-only")
	asrt.NotContains(encoded, "=", "base45 alphabet does not include =")

	decoded, err := DecodeSDP(encoded)
	require.NoError(t, err)
	asrt.Equal(input, decoded)
}

func Test_DecodeSDPErrors(t *testing.T) {
	asrt := assert.New(t)

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		// lowercase fails base45 decode
		{"invalid base45 alphabet", "hello world"},
		// RFC 9285 vector: decodes to "Hello!!"
		{"base45 but not zstd", "%69 VD92EX0"},
	}

	for _, cur := range tests {
		_, err := DecodeSDP(cur.input)
		asrt.Error(err, cur.name)
	}
}

// encodeSDPFixture returns a valid encoded SDP string for use as test input.
// It round-trips a minimal SessionDescription through EncodeSDP so the test
// does not depend on any particular encoder output format.
func encodeSDPFixture(t *testing.T) string {
	t.Helper()
	desc := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "v=0\r\n"}
	encoded, err := EncodeSDP(desc)
	if err != nil {
		t.Fatalf("EncodeSDP: %v", err)
	}
	return encoded
}

func TestResolveSDPFlag_Literal(t *testing.T) {
	encoded := encodeSDPFixture(t)
	got, err := ResolveSDPFlag(encoded, strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != encoded {
		t.Fatalf("got %q, want %q", got, encoded)
	}
}

func TestResolveSDP_AtFile(t *testing.T) {
	encoded := encodeSDPFixture(t)
	path := filepath.Join(t.TempDir(), "answer.txt")
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ResolveSDPFlag("@"+path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != encoded {
		t.Fatalf("got %q, want %q", got, encoded)
	}
}

func TestResolveSDPFlag_AtStdin(t *testing.T) {
	encoded := encodeSDPFixture(t)
	got, err := ResolveSDPFlag("@-", strings.NewReader(encoded))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != encoded {
		t.Fatalf("got %q, want %q", got, encoded)
	}
}

func TestResolveSDP_TrimsTrailingNewline(t *testing.T) {
	encoded := encodeSDPFixture(t)
	path := filepath.Join(t.TempDir(), "answer.txt")
	if err := os.WriteFile(path, []byte(encoded+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ResolveSDPFlag("@"+path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != encoded {
		t.Fatalf("got %q, want %q (trailing whitespace not trimmed)", got, encoded)
	}
}

func TestResolveSDPFlag_LiteralInvalid(t *testing.T) {
	_, err := ResolveSDPFlag("not-a-real-sdp", strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid SDP value:") {
		t.Fatalf("error %q does not contain %q", err.Error(), "invalid SDP value:")
	}
}

func TestResolveSDPFlag_FileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.txt")
	_, err := ResolveSDPFlag("@"+path, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read SDP source:") {
		t.Fatalf("error %q does not contain %q", err.Error(), "read SDP source:")
	}
}

func TestResolveSDP_EmptyResolved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := ResolveSDPFlag("@"+path, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid SDP value: empty SDP") {
		t.Fatalf("error %q does not contain %q", err.Error(), "invalid SDP value: empty SDP")
	}
}
