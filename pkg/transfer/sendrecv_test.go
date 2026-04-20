package transfer_test

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonito/gfile/internal/output"
	"github.com/antonito/gfile/pkg/transfer"
	"github.com/antonito/gfile/pkg/transfer/receiver"
	"github.com/antonito/gfile/pkg/transfer/sender"
)

// runTransfer drives a full sender+receiver loop in-process. Returns when
// both sides finish. Fails the test on any error, timeout, or output
// mismatch.
func runTransfer(t *testing.T, payload []byte) {
	runTransferAt(t, payload, 0)
}

// runTransferAt drives a transfer with the given sender compression level.
func runTransferAt(t *testing.T, payload []byte, compressionLevel int) {
	t.Helper()
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.bin")
	outPath := filepath.Join(dir, "out.bin")
	require.NoError(t, os.WriteFile(inPath, payload, 0o600), "seed")
	in, err := os.Open(inPath)
	require.NoError(t, err, "open in")
	defer in.Close()
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	require.NoError(t, err, "open out")
	defer out.Close()

	offerR, offerW := io.Pipe()
	answerR, answerW := io.Pipe()

	snd, err := sender.NewWith(sender.Config{
		IOConfig: transfer.IOConfig{
			SDPProvider: answerR,
			SDPOutput:   offerW,
			DisableQR:   true,
			DisableMDNS: true,
			ICELite:     true,
		},
		Stream:           in,
		Connections:      1,
		CompressionLevel: compressionLevel,
	})
	require.NoError(t, err, "sender NewWith")
	rcv := receiver.NewWith(receiver.Config{
		IOConfig: transfer.IOConfig{
			SDPProvider: offerR,
			SDPOutput:   answerW,
			DisableQR:   true,
			DisableMDNS: true,
			ICELite:     true,
		},
		Stream: out,
		Path:   outPath,
	})

	senderDone := make(chan error, 1)
	recvDone := make(chan error, 1)
	go func() {
		senderDone <- snd.Start()
	}()
	go func() {
		recvDone <- rcv.Start(t.Context())
	}()

	select {
	case err := <-senderDone:
		require.NoError(t, err, "sender")
	case <-time.After(60 * time.Second):
		t.Fatal("sender timeout")
	}
	select {
	case err := <-recvDone:
		require.NoError(t, err, "receiver")
	case <-time.After(60 * time.Second):
		t.Fatal("receiver timeout")
	}

	got, err := os.ReadFile(outPath)
	require.NoError(t, err, "read out")
	require.Truef(t, bytes.Equal(got, payload), "output mismatch: len=%d want=%d", len(got), len(payload))
}

func TestSendReceiveSmall(t *testing.T) {
	payload := []byte("hello, gfile — one-line test payload")
	runTransfer(t, payload)
}

func TestSendReceiveOneChunk(t *testing.T) {
	payload := make([]byte, 16*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err, "rand")
	runTransfer(t, payload)
}

func TestSendReceiveMultiChunk(t *testing.T) {
	payload := make([]byte, 5*1024*1024) // 5 MB
	_, err := rand.Read(payload)
	require.NoError(t, err, "rand")
	runTransfer(t, payload)
}

func TestSendReceiveEmptyFile(t *testing.T) {
	runTransfer(t, nil)
}

func TestSendReceiveCompressed(t *testing.T) {
	// Repetitive payload across several chunks so compression actually
	// engages and we exercise multiple DATA frames under CodecZstd.
	const size = 3 * 1024 * 1024
	payload := make([]byte, size)
	for ndx := range payload {
		payload[ndx] = byte(ndx % 11)
	}
	runTransferAt(t, payload, 1)
}

func TestSendReceiveCompressedRandomInput(t *testing.T) {
	// Random bytes defeat compression; zstd's incompressible fallback
	// should still produce a valid frame that the receiver decodes to the
	// original bytes.
	payload := make([]byte, 512*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err, "rand")
	runTransferAt(t, payload, 1)
}

func TestSendReceiveMultiSetupTimeout(t *testing.T) {
	// Force negotiatePeers to abort almost immediately by using a 1ns
	// override. This exercises the ABORT path end-to-end.
	sender.SetPeerSetupTimeoutForTest(1 * time.Nanosecond)
	defer sender.SetPeerSetupTimeoutForTest(0)

	payload := []byte("x")
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.bin")
	outPath := filepath.Join(dir, "out.bin")
	require.NoError(t, os.WriteFile(inPath, payload, 0o600), "seed")
	in, err := os.Open(inPath)
	require.NoError(t, err, "open in")
	defer in.Close()
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	require.NoError(t, err, "open out")
	defer out.Close()

	offerR, offerW := io.Pipe()
	answerR, answerW := io.Pipe()

	snd, err := sender.NewWith(sender.Config{
		IOConfig: transfer.IOConfig{
			SDPProvider: answerR,
			SDPOutput:   offerW,
			DisableQR:   true,
			DisableMDNS: true,
			ICELite:     true,
		},
		Stream:      in,
		Connections: 2,
	})
	require.NoError(t, err, "sender NewWith")
	rcv := receiver.NewWith(receiver.Config{
		IOConfig: transfer.IOConfig{
			SDPProvider: offerR,
			SDPOutput:   answerW,
			DisableQR:   true,
			DisableMDNS: true,
			ICELite:     true,
		},
		Stream: out,
		Path:   outPath,
	})

	senderDone := make(chan error, 1)
	recvDone := make(chan error, 1)
	go func() {
		senderDone <- snd.Start()
	}()
	go func() {
		recvDone <- rcv.Start(t.Context())
	}()

	select {
	case <-senderDone:
		// Sender's Start() returns nil on our normal teardown path; the
		// important thing is that it returned, not that it errored.
	case <-time.After(30 * time.Second):
		t.Fatal("sender did not exit within 30s")
	}
	select {
	case err := <-recvDone:
		require.Error(t, err, "receiver should have errored after sender ABORT")
	case <-time.After(30 * time.Second):
		t.Fatal("receiver did not exit after sender abort")
	}
	_, stat := os.Stat(outPath)
	assert.True(t, os.IsNotExist(stat), "partial output not cleaned up after abort")
}

func TestSendReceiveReceiverProgressEmitted(t *testing.T) {
	// Swap the package-global Default emitter for one we can read back
	// JSON progress events from. We only assert on the receiver's events.
	var captured bytes.Buffer
	restore := output.SetDefault(output.New(output.ModeJSON, &captured, io.Discard))
	defer restore()

	// 5 MB payload: large enough that the progress goroutine is active
	// during the transfer. The initial upfront emit guarantees at least one
	// progress event even on loopback machines that finish in under 100ms.
	payload := make([]byte, 5*1024*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err, "rand")
	runTransfer(t, payload)

	scanner := bufio.NewScanner(&captured)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var sawReceiverProgress bool
	for scanner.Scan() {
		var ev struct {
			Type string `json:"type"`
			Role string `json:"role"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // non-JSON or unrelated line
		}
		if ev.Type == "progress" && ev.Role == "receiver" {
			sawReceiverProgress = true
			break
		}
	}
	require.Truef(t, sawReceiverProgress,
		"expected at least one progress event with role=receiver; captured:\n%s",
		captured.String())
}
