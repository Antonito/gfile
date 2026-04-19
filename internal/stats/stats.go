package stats

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Stats tracks transfer statistics (bytes, duration, bandwidth).
type Stats struct {
	nbBytes atomic.Uint64

	mu         sync.Mutex
	timeStart  time.Time
	timeStop   time.Time
	timePause  time.Time
	timePaused time.Duration

	// nowFn is the time source. Always non-nil (defaulted in New, restored
	// by setClockForTest(nil)). Tests inject a deterministic clock.
	nowFn func() time.Time

	ring bwRing
}

// New returns a zero-valued Stats whose time source is time.Now.
func New() *Stats {
	return &Stats{nowFn: time.Now}
}

// setClockForTest overrides the time source. Pass nil to restore time.Now.
func (s *Stats) setClockForTest(fn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fn == nil {
		s.nowFn = time.Now
	} else {
		s.nowFn = fn
	}
}

func (s *Stats) now() time.Time {
	return s.nowFn()
}

// Bytes returns the running byte total.
func (s *Stats) Bytes() uint64 {
	return s.nbBytes.Load()
}

// AddBytes adds count to the running byte total.
func (s *Stats) AddBytes(count uint64) {
	s.nbBytes.Add(count)
}

// Start records the start timestamp on the first call, or resumes a
// paused run. Subsequent calls while running are no-ops.
func (s *Stats) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timeStart.IsZero() {
		s.timeStart = s.now()
		return
	}
	if !s.timePause.IsZero() {
		s.timePaused += s.now().Sub(s.timePause)
		s.timePause = time.Time{}
	}
}

// Pause records an interruption timestamp.
// No-op if not started or already stopped.
func (s *Stats) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timeStart.IsZero() || !s.timeStop.IsZero() {
		return
	}
	if s.timePause.IsZero() {
		s.timePause = s.now()
	}
}

// Stop records the stop timestamp.
// No-op if not started or already stopped.
func (s *Stats) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timeStart.IsZero() {
		return
	}
	if s.timeStop.IsZero() {
		s.timeStop = s.now()
	}
}

// Duration returns the active elapsed time, excluding any intervals
// spent paused. Returns 0 before Start. While running, measures up to
// "now"; after Stop, up to the stop timestamp.
func (s *Stats) Duration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.durationLocked()
}

func (s *Stats) durationLocked() time.Duration {
	if s.timeStart.IsZero() {
		return 0
	}

	if s.timeStop.IsZero() {
		return s.now().Sub(s.timeStart) - s.timePaused
	}

	return s.timeStop.Sub(s.timeStart) - s.timePaused
}

func (s *Stats) String() string {
	s.mu.Lock()
	elapsed := s.durationLocked()
	s.mu.Unlock()

	totalBytes := s.nbBytes.Load()
	var bandwidth float64
	if elapsed > 0 {
		bandwidth = (float64(totalBytes) / 1024 / 1024) / elapsed.Seconds()
	}

	return fmt.Sprintf("%v bytes | %-v | %0.4f MB/s", totalBytes, elapsed, bandwidth)
}
