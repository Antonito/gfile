package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/antonito/gfile/internal/stats"
)

// Mode selects the Emitter's output format.
type Mode int

const (
	// ModeText emits human-readable text.
	ModeText Mode = iota

	// ModeJSON emits one JSON event per line for external tools.
	ModeJSON
)

const progressMinInterval = 250 * time.Millisecond

// Emitter serialises CLI emissions (text or JSON)
type Emitter struct {
	mu           sync.Mutex
	mode         Mode
	stdout       io.Writer
	stderr       io.Writer
	progressLast map[string]time.Time
	clock        func() time.Time
}

// New constructs an Emitter with the given mode and writers
func New(
	mode Mode,
	stdout io.Writer,
	stderr io.Writer,
) *Emitter {
	return &Emitter{
		mode:         mode,
		stdout:       stdout,
		stderr:       stderr,
		progressLast: map[string]time.Time{},
		clock:        time.Now,
	}
}

// SetMode switches the output mode.
func (e *Emitter) SetMode(mode Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.mode = mode
}

// CurrentMode returns the active output mode.
func (e *Emitter) CurrentMode() Mode {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.mode
}

// Prompt writes an interactive message to stderr.
// No-op in JSON mode.
func (e *Emitter) Prompt(msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.mode == ModeJSON {
		return
	}

	_, _ = fmt.Fprintln(e.stderr, msg)
}

// BenchTotal writes the pre-transfer total byte count.
//
// JSON mode emits a `bench_total` event so consumers can compute
// a fill ratio (progress events only report current bytes, not the target).
func (e *Emitter) BenchTotal(bytes int64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.mode == ModeJSON {
		_ = json.NewEncoder(e.stdout).Encode(struct {
			Type  string `json:"type"`
			Bytes int64  `json:"bytes"`
		}{"bench_total", bytes})
		return
	}

	_, _ = fmt.Fprintf(e.stdout, "Bench total: %d bytes\n", bytes)
}

// TransferComplete emits a one-line confirmation that the transfer
// finished successfully, identifying the file and its size on disk.
//
// JSON mode emits a `transfer_complete` event so external consumers
// can detect completion without parsing text.
func (e *Emitter) TransferComplete(
	role string,
	path string,
	bytes int64,
) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.mode == ModeJSON {
		_ = json.NewEncoder(e.stdout).Encode(struct {
			Type  string `json:"type"`
			Role  string `json:"role"`
			Path  string `json:"path"`
			Bytes int64  `json:"bytes"`
		}{"transfer_complete", role, path, bytes})
		return
	}

	verb := "Received"
	if role == "sender" {
		verb = "Sent"
	}
	_, _ = fmt.Fprintf(e.stdout, "%s: %s (%s)\n", verb, path, humanBytes(bytes))
}

// humanBytes renders a byte count using 1024-based units
// (matches the MB/s convention used elsewhere in the CLI).
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Stats writes per-side final transfer statistics
func (e *Emitter) Stats(
	role string,
	sts *stats.Stats,
) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.mode == ModeJSON {
		totalBytes := sts.Bytes()
		elapsed := sts.Duration()
		var bps float64
		if elapsed > 0 {
			bps = float64(totalBytes) / elapsed.Seconds()
		}
		_ = json.NewEncoder(e.stdout).Encode(struct {
			Type        string  `json:"type"`
			Role        string  `json:"role"`
			Bytes       uint64  `json:"bytes"`
			DurationNS  int64   `json:"duration_ns"`
			BytesPerSec float64 `json:"bytes_per_sec"`
		}{"stats", role, totalBytes, elapsed.Nanoseconds(), bps})
		return
	}

	prefix := "Download: "
	if role == "sender" {
		prefix = "Upload:   "
	}
	_, _ = fmt.Fprintf(e.stdout, "%s%s\n", prefix, sts.String())
}

// SDP writes the local session description.
func (e *Emitter) SDP(
	writer io.Writer,
	role string,
	sdp string,
) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.mode == ModeJSON {
		_ = json.NewEncoder(writer).Encode(struct {
			Type string `json:"type"`
			Role string `json:"role"`
			SDP  string `json:"sdp"`
		}{"sdp", role, sdp})
		return
	}

	_, _ = fmt.Fprintln(e.stderr, "Send this SDP:")
	_, _ = fmt.Fprintln(writer, sdp)
}

// Progress emits a transfer progress event.
//
// Throttled per-role to a minimum interval (progressMinInterval) so neither
// the terminal nor the JSON stream gets flooded.
func (e *Emitter) Progress(
	role string,
	bytes int64,
	bytesPerSec float64,
) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := e.clock()
	if last, ok := e.progressLast[role]; ok && now.Sub(last) < progressMinInterval {
		return
	}
	e.progressLast[role] = now

	if e.mode == ModeJSON {
		_ = json.NewEncoder(e.stdout).Encode(struct {
			Type        string  `json:"type"`
			Role        string  `json:"role"`
			Bytes       int64   `json:"bytes"`
			BytesPerSec float64 `json:"bytes_per_sec"`
		}{"progress", role, bytes, bytesPerSec})
		return
	}

	mib := bytesPerSec / (1024 * 1024)
	_, _ = fmt.Fprintf(e.stderr, "Transferring at %.2f MB/s (%d bytes)\r", mib, bytes)
}

type kindedError interface {
	Kind() string
}

// Fatal reports a fatal error before non-zero process exit.
func (e *Emitter) Fatal(
	err error,
	kindDefault string,
) {
	if err == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	kind := kindDefault
	var ke kindedError
	if errors.As(err, &ke) {
		kind = ke.Kind()
	}

	if e.mode == ModeJSON {
		_ = json.NewEncoder(e.stdout).Encode(struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Kind    string `json:"kind"`
		}{"error", err.Error(), kind})
		return
	}
	_, _ = fmt.Fprintln(e.stderr, err)
}
