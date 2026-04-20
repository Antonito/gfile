package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonito/gfile/internal/stats"
)

func TestModeRoundTrip(t *testing.T) {
	em := New(ModeText, io.Discard, io.Discard)

	require.Equal(t, ModeText, em.CurrentMode(), "default mode")
	em.SetMode(ModeJSON)
	require.Equal(t, ModeJSON, em.CurrentMode(), "after SetMode(ModeJSON)")
	em.SetMode(ModeText)
	require.Equal(t, ModeText, em.CurrentMode(), "after SetMode(ModeText)")
}

func TestPromptText(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeText, &out, &errBuf)

	em.Prompt("Please, paste the remote SDP:")

	asrt.Empty(out.String(), "stdout")
	asrt.Equal("Please, paste the remote SDP:\n", errBuf.String(), "stderr")
}

func TestPromptJSONIsNoOp(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	em.Prompt("Send this SDP:")

	asrt.Empty(out.String(), "stdout")
	asrt.Empty(errBuf.String(), "stderr")
}

func TestBenchTotalText(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeText, &out, &errBuf)

	em.BenchTotal(524288000)

	asrt.Equal("Bench total: 524288000 bytes\n", out.String(), "stdout")
	asrt.Empty(errBuf.String(), "stderr")
}

func TestBenchTotalJSON(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	em.BenchTotal(524288000)

	asrt.Empty(errBuf.String(), "stderr")
	var ev struct {
		Type  string `json:"type"`
		Bytes int64  `json:"bytes"`
	}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &ev), "unmarshal %q", out.String())
	asrt.Equal("bench_total", ev.Type)
	asrt.Equal(int64(524288000), ev.Bytes)
}

func newStoppedStats(t *testing.T, totalBytes uint64, duration time.Duration) *stats.Stats {
	t.Helper()
	sts := stats.New()
	sts.AddBytes(totalBytes)
	sts.Start()

	time.Sleep(duration)
	sts.Stop()

	return sts
}

func TestStatsTextSender(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeText, &out, &errBuf)

	sts := newStoppedStats(t, 1024, 1*time.Millisecond)
	em.Stats("sender", sts)

	got := out.String()
	asrt.True(strings.HasPrefix(got, "Upload:   "), "want `Upload:   ` prefix, got %q", got)
	asrt.Contains(got, "1024 bytes")
}

func TestStatsTextReceiver(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeText, &out, &errBuf)

	sts := newStoppedStats(t, 2048, 1*time.Millisecond)
	em.Stats("receiver", sts)

	got := out.String()
	asrt.True(strings.HasPrefix(got, "Download: "), "want `Download: ` prefix, got %q", got)
}

func TestStatsJSON(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	sts := newStoppedStats(t, 524288000, 1*time.Millisecond)
	em.Stats("sender", sts)

	var ev struct {
		Type        string  `json:"type"`
		Role        string  `json:"role"`
		Bytes       int64   `json:"bytes"`
		DurationNS  int64   `json:"duration_ns"`
		BytesPerSec float64 `json:"bytes_per_sec"`
	}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &ev), "unmarshal %q", out.String())
	asrt.Equal("stats", ev.Type)
	asrt.Equal("sender", ev.Role)
	asrt.Equal(int64(524288000), ev.Bytes)
	asrt.Positive(ev.DurationNS)
	asrt.Positive(ev.BytesPerSec)
}

func TestProgressTextThrottle(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeText, &out, &errBuf)

	var fakeNow time.Time
	em.clock = func() time.Time { return fakeNow }

	// first call: emits
	fakeNow = time.Unix(0, 0)
	em.Progress("sender", 1000, 100.0)

	// too soon: throttled
	fakeNow = fakeNow.Add(100 * time.Millisecond)
	em.Progress("sender", 2000, 200.0)

	// now 300ms after first -> emits
	fakeNow = fakeNow.Add(200 * time.Millisecond)
	em.Progress("sender", 3000, 300.0)

	got := errBuf.String()
	asrt.Equal(2, strings.Count(got, "Transferring at"), "want exactly 2 emissions in stderr; got %q", got)
	asrt.Empty(out.String(), "stdout should be empty in text mode")
}

func TestProgressJSONThrottle(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	var fakeNow time.Time
	em.clock = func() time.Time { return fakeNow }

	// emits -> throttled -> emits
	fakeNow = time.Unix(0, 0)
	em.Progress("sender", 1000, 100.0)
	fakeNow = fakeNow.Add(100 * time.Millisecond)
	em.Progress("sender", 2000, 200.0)
	fakeNow = fakeNow.Add(200 * time.Millisecond)
	em.Progress("sender", 3000, 300.0)

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 2, "want 2 JSON events, got %q", out.String())
	var ev struct {
		Type        string  `json:"type"`
		Role        string  `json:"role"`
		Bytes       int64   `json:"bytes"`
		BytesPerSec float64 `json:"bytes_per_sec"`
	}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &ev), "first event")
	asrt.Equal("progress", ev.Type)
	asrt.Equal("sender", ev.Role)
	asrt.Equal(int64(1000), ev.Bytes)
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &ev), "second event")
	asrt.Equal(int64(3000), ev.Bytes)
}

func TestProgressPerRoleThrottle(t *testing.T) {
	// Throttle is per-role: sender and receiver don't share the same budget.
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	var fakeNow time.Time
	em.clock = func() time.Time { return fakeNow }

	// emits -> emits (different role) -> throttled (same role)
	fakeNow = time.Unix(0, 0)
	em.Progress("sender", 1000, 100.0)
	fakeNow = fakeNow.Add(10 * time.Millisecond)
	em.Progress("receiver", 2000, 200.0)
	fakeNow = fakeNow.Add(10 * time.Millisecond)
	em.Progress("sender", 3000, 300.0)

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	require.Len(t, lines, 2, "want 2 events (one per role, third throttled); got %q", out.String())
}

func TestSDPText(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeText, &out, &errBuf)

	var sdpOut bytes.Buffer
	em.SDP(&sdpOut, "sender", "eyJ0eXAi...")

	asrt.Equal("Send this SDP:\n", errBuf.String(), "stderr label")
	asrt.Equal("eyJ0eXAi...\n", sdpOut.String(), "sdp writer")
	asrt.Empty(out.String(), "emitter stdout should be untouched in text mode")
}

func TestSDPJSON(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	var sdpOut bytes.Buffer
	em.SDP(&sdpOut, "receiver", "abc123")

	asrt.Empty(errBuf.String(), "stderr should be empty in JSON mode")
	var ev struct {
		Type string `json:"type"`
		Role string `json:"role"`
		SDP  string `json:"sdp"`
	}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(sdpOut.String())), &ev), "unmarshal %q", sdpOut.String())
	asrt.Equal("sdp", ev.Type)
	asrt.Equal("receiver", ev.Role)
	asrt.Equal("abc123", ev.SDP)
}

type fakeKindedErr struct{ kind string }

func (e fakeKindedErr) Error() string { return "boom" }
func (e fakeKindedErr) Kind() string  { return e.kind }

func TestFatalTextWritesStderr(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeText, &out, &errBuf)

	em.Fatal(errors.New("boom"), "internal")

	asrt.Empty(out.String(), "stdout should be empty in text mode")
	asrt.Contains(errBuf.String(), "boom", "stderr should contain error message")
}

func TestFatalJSONWritesEvent(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	em.Fatal(errors.New("sctp: association closed"), "transport")

	var ev struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Kind    string `json:"kind"`
	}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &ev), "unmarshal %q", out.String())
	asrt.Equal("error", ev.Type)
	asrt.Equal("sctp: association closed", ev.Message)
	asrt.Equal("transport", ev.Kind)
}

// slowLockedWriter splits each Write into two halves
// with a yield between them.
type slowLockedWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *slowLockedWriter) Write(buf []byte) (int, error) {
	if len(buf) < 2 {
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.buf.Write(buf)
	}

	w.mu.Lock()
	n1, err := w.buf.Write(buf[:len(buf)/2])
	w.mu.Unlock()

	if err != nil {
		return n1, err
	}

	for range 10 {
		runtime.Gosched()
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	n2, err := w.buf.Write(buf[len(buf)/2:])

	return n1 + n2, err
}

func (w *slowLockedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// TestEmittersSerializeConcurrentWrites pins the guarantee that concurrent
// emitters never interleave bytes on the shared stdout/stderr writers.
//
// Uses a slow writer that forces interleaving if emitters don't serialize
// their Encode calls themselves.
func TestEmittersSerializeConcurrentWrites(t *testing.T) {
	asrt := assert.New(t)
	slow := &slowLockedWriter{}
	em := New(ModeJSON, slow, io.Discard)

	const N = 100
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(N)
	for ndx := range N {
		go func(ndx int) {
			defer wg.Done()
			<-start
			em.BenchTotal(int64(ndx))
		}(ndx)
	}
	close(start)
	wg.Wait()

	scanner := bufio.NewScanner(strings.NewReader(slow.String()))
	seen := 0
	for scanner.Scan() {
		var ev struct {
			Type  string `json:"type"`
			Bytes int64  `json:"bytes"`
		}
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &ev), "line %d malformed (emitter writes interleaved): %q", seen, scanner.Bytes())
		asrt.Equal("bench_total", ev.Type, "line %d", seen)
		seen++
	}
	require.NoError(t, scanner.Err(), "scan")
	asrt.Equal(N, seen)
}

func TestFatalJSONUsesKindedErrorInterface(t *testing.T) {
	asrt := assert.New(t)
	var out, errBuf bytes.Buffer
	em := New(ModeJSON, &out, &errBuf)

	em.Fatal(fakeKindedErr{kind: "config"}, "internal")

	var ev struct {
		Kind string `json:"kind"`
	}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &ev), "unmarshal")
	asrt.Equal("config", ev.Kind, "error interface should win")
}

func TestSetDefaultRestores(t *testing.T) {
	var out bytes.Buffer
	custom := New(ModeJSON, &out, io.Discard)

	orig := Default
	restore := SetDefault(custom)
	require.Equal(t, custom, Default, "Default not swapped to custom emitter")
	restore()
	require.Equal(t, orig, Default, "Default not restored after restore()")
}
