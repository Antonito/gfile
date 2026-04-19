package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_CurrentBandwidth_NotStarted(t *testing.T) {
	asrt := assert.New(t)
	sts := New()
	// No Start, no Sample.
	asrt.InDelta(0.0, sts.CurrentBandwidth(), 1e-9)
}

func Test_CurrentBandwidth_Startup(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := t0
	sts.setClockForTest(func() time.Time { return now })

	sts.Start()
	// 100ms in, 100KB transferred → 1 MB/s.
	now = t0.Add(100 * time.Millisecond)
	sts.AddBytes(100 * 1024)
	sts.Sample()

	got := sts.CurrentBandwidth()
	want := float64(100*1024) / 0.1 // bytes per second
	asrt.InDelta(want, got, 1.0)

	sts.setClockForTest(nil)
}

func Test_CurrentBandwidth_SteadyState(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := t0
	sts.setClockForTest(func() time.Time { return now })

	sts.Start()
	// 10s of transfer at exactly 10 MB/s, sampled every 100ms.
	const ratePerTick = uint64(10 * 1024 * 1024 / 10) // bytes per 100ms
	for ndx := 1; ndx <= 100; ndx++ {
		now = t0.Add(time.Duration(ndx) * 100 * time.Millisecond)
		sts.AddBytes(ratePerTick)
		sts.Sample()
	}

	got := sts.CurrentBandwidth()
	want := 10.0 * 1024 * 1024
	asrt.InDelta(want, got, want*0.001)

	sts.setClockForTest(nil)
}

func Test_CurrentBandwidth_StallDecaysToZero(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := t0
	sts.setClockForTest(func() time.Time { return now })

	sts.Start()
	const ratePerTick = uint64(10 * 1024 * 1024 / 10)
	for ndx := 1; ndx <= 60; ndx++ {
		now = t0.Add(time.Duration(ndx) * 100 * time.Millisecond)
		sts.AddBytes(ratePerTick)
		sts.Sample()
	}
	asrt.InDelta(10.0*1024*1024, sts.CurrentBandwidth(), 0.05*10*1024*1024)

	for ndx := 61; ndx <= 120; ndx++ {
		now = t0.Add(time.Duration(ndx) * 100 * time.Millisecond)
		sts.Sample() // no AddBytes
	}
	asrt.InDelta(0.0, sts.CurrentBandwidth(), 1e-9)

	sts.setClockForTest(nil)
}

func Test_CurrentBandwidth_RingBufferBounded(t *testing.T) {
	asrt := assert.New(t)
	sts := New()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := t0
	sts.setClockForTest(func() time.Time { return now })

	sts.Start()
	for ndx := 1; ndx <= 1000; ndx++ {
		now = t0.Add(time.Duration(ndx) * 100 * time.Millisecond)
		sts.AddBytes(1024)
		sts.Sample()
	}
	got := sts.CurrentBandwidth()
	want := float64(10 * 1024)
	asrt.InDelta(want, got, want*0.05)
	asrt.Equal(maxWindowSamples, sts.ring.count)

	sts.setClockForTest(nil)
}
