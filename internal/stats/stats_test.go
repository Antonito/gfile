package stats

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Bytes(t *testing.T) {
	asrt := assert.New(t)

	tests := []struct {
		before uint64
		add    uint64
		after  uint64
	}{
		{before: 0, add: 0, after: 0},
		{before: 0, add: 1, after: 1},
		{before: 1, add: 10, after: 11},
	}

	sts := New()
	for _, cur := range tests {
		asrt.Equal(cur.before, sts.Bytes())
		sts.AddBytes(cur.add)
		asrt.Equal(cur.after, sts.Bytes())
	}
}

func Test_ControlFlow(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	// Everything should be 0 at the beginning.
	asrt.True(sts.timeStart.IsZero())
	asrt.True(sts.timeStop.IsZero())
	asrt.True(sts.timePause.IsZero())

	// Should not do anything.
	sts.Stop()
	asrt.True(sts.timeStop.IsZero())

	sts.Pause()
	asrt.True(sts.timePause.IsZero())

	// Should start.
	sts.Start()
	originalStart := sts.timeStart
	asrt.False(sts.timeStart.IsZero())

	// Should pause.
	sts.Pause()
	asrt.False(sts.timePause.IsZero())
	originalPause := sts.timePause
	// Should not modify when already paused.
	sts.Pause()
	asrt.Equal(originalPause, sts.timePause)

	// Should resume.
	asrt.Equal(int64(0), sts.timePaused.Nanoseconds())
	sts.Start()
	asrt.NotEqual(0, sts.timePaused.Nanoseconds())
	originalPausedDuration := sts.timePaused
	asrt.True(sts.timePause.IsZero())
	asrt.Equal(originalStart, sts.timeStart)

	sts.Pause()
	time.Sleep(10 * time.Nanosecond)
	sts.Start()
	asrt.Greater(sts.timePaused, originalPausedDuration)

	sts.Stop()
	asrt.False(sts.timeStop.IsZero())
}

func Test_Bandwidth(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	now := time.Now()
	tests := []struct {
		startTime         time.Time
		stopTime          time.Time
		pauseDuration     time.Duration
		bytesCount        uint64
		expectedBandwidth float64
	}{
		{
			// Not started: Duration is 0, Bandwidth returns 0 (not NaN).
			startTime:         time.Time{},
			stopTime:          time.Time{},
			pauseDuration:     0,
			bytesCount:        0,
			expectedBandwidth: 0,
		},
		{
			startTime:         now,
			stopTime:          time.Time{},
			pauseDuration:     0,
			bytesCount:        0,
			expectedBandwidth: 0,
		},
		{
			startTime:         now,
			stopTime:          now.Add(time.Second),
			pauseDuration:     0,
			bytesCount:        1024 * 1024,
			expectedBandwidth: 1,
		},
		{
			startTime:         now,
			stopTime:          now.Add(2 * time.Second),
			pauseDuration:     time.Second,
			bytesCount:        1024 * 1024,
			expectedBandwidth: 1,
		},
	}

	for _, cur := range tests {
		sts.timeStart = cur.startTime
		sts.timeStop = cur.stopTime
		sts.timePaused = cur.pauseDuration
		sts.nbBytes.Store(cur.bytesCount)

		if math.IsNaN(cur.expectedBandwidth) {
			asrt.True(math.IsNaN(sts.bandwidth()))
		} else {
			asrt.InDelta(cur.expectedBandwidth, sts.bandwidth(), 1e-9)
		}
	}
}

func Test_Duration(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	// Should be 0 before Start.
	asrt.Equal(time.Duration(0), sts.Duration())

	// Should return time.Since() once running.
	sts.Start()
	durationTmp := sts.Duration()
	time.Sleep(10 * time.Nanosecond)
	asrt.Greater(sts.Duration(), durationTmp)
}

func Test_ClockInjection(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := t0
	sts.setClockForTest(func() time.Time { return now })

	sts.Start()
	now = t0.Add(2 * time.Second)
	asrt.Equal(2*time.Second, sts.Duration())

	sts.setClockForTest(nil) // restore
}
