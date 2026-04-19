package stats

import (
	"sync"
	"time"
)

// maxWindow caps the live-bandwidth averaging window. Past this, the oldest
// sample ages out and the window slides.
const maxWindow = 5 * time.Second

// sampleTickInterval is the cadence at which callers are expected to invoke
// Sample; the ring buffer is sized for one tick per (maxWindow / interval).
// Off-cadence calls still work, just with fewer or more recent samples.
const sampleTickInterval = 100 * time.Millisecond

const maxWindowSamples = int(maxWindow / sampleTickInterval) // = 50

// bwSample is one ring-buffer entry.
type bwSample struct {
	t     time.Time
	bytes uint64
}

// bwRing is a fixed-capacity ring buffer of (time, bytes) samples used by
// CurrentBandwidth. Its own mutex keeps it independent of Stats.mu so live
// progress reads don't serialize against Duration() callers.
type bwRing struct {
	mu      sync.Mutex
	samples [maxWindowSamples]bwSample
	head    int // index of the next slot to write
	count   int // number of valid entries (0..maxWindowSamples)
}

func (r *bwRing) push(sample bwSample) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.samples[r.head] = sample
	r.head = (r.head + 1) % maxWindowSamples

	if r.count < maxWindowSamples {
		r.count++
	}
}

// oldestWithin returns the oldest sample with timestamp >= cutoff; ok=false
// if no sample falls in the window.
func (r *bwRing) oldestWithin(cutoff time.Time) (bwSample, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == 0 {
		return bwSample{}, false
	}

	// Oldest index = head - count (mod cap).
	start := (r.head - r.count + maxWindowSamples) % maxWindowSamples
	for ndx := range r.count {
		idx := (start + ndx) % maxWindowSamples
		if !r.samples[idx].t.Before(cutoff) {
			return r.samples[idx], true
		}
	}

	return bwSample{}, false
}

// bandwidth returns the average throughput in MB/s across the full
// active duration (Start to now or Stop, minus pauses). For a
// responsive live rate over the last maxWindow, use CurrentBandwidth.
func (s *Stats) bandwidth() float64 {
	elapsed := s.Duration()
	if elapsed == 0 {
		return 0
	}

	return (float64(s.nbBytes.Load()) / 1024 / 1024) / elapsed.Seconds()
}

// Sample records the current byte total at "now" into the live-bandwidth
// ring buffer. Cheap; intended to be called on a regular tick (e.g. every
// 100ms by progress emitters).
func (s *Stats) Sample() {
	// Snapshot the clock under s.mu: now() reads s.nowFn which is guarded
	// by s.mu (see setClockForTest), otherwise the race detector trips.
	s.mu.Lock()
	currentTime := s.now()
	s.mu.Unlock()

	s.ring.push(bwSample{t: currentTime, bytes: s.nbBytes.Load()})
}

// CurrentBandwidth returns bytes/sec over an adaptive window:
//
//	window_age = min(time_since_oldest_in_window_sample, maxWindow)
//	rate       = (bytes_now − bytes_at_window_start) / window_age
//
// During the first maxWindow, Stats.timeStart acts as an implicit zeroth
// sample at bytes=0, so the rate is meaningful from the first AddBytes
// call without a startup spike.
//
// Returns 0 if not started, or past the startup window with no recent samples.
func (s *Stats) CurrentBandwidth() float64 {
	// Snapshot start AND now under s.mu (now reads s.nowFn).
	s.mu.Lock()
	start := s.timeStart
	now := s.now()
	s.mu.Unlock()

	if start.IsZero() {
		return 0
	}

	cutoff := now.Add(-maxWindow)
	bytesNow := s.nbBytes.Load()

	// Prefer timeStart as the zeroth sample while it is still within
	// maxWindow; otherwise fall back to the oldest real sample.
	var (
		refTime  time.Time
		refBytes uint64
	)

	if !start.Before(cutoff) {
		refTime = start
		refBytes = 0
	} else if oldest, ok := s.ring.oldestWithin(cutoff); ok {
		refTime = oldest.t
		refBytes = oldest.bytes
	} else {
		// Started long ago, but no recent samples (shouldn't happen if the
		// caller ticks at 100ms). Return 0 rather than guess.
		return 0
	}

	dt := now.Sub(refTime).Seconds()
	if dt <= 0 {
		return 0
	}

	return float64(bytesNow-refBytes) / dt
}
