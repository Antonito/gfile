package stats

import (
	"fmt"
	"time"
)

// Stats provide a way to track statistics infos
type Stats struct {
	nbBytes   uint64
	timeStart time.Time
	timeStop  time.Time

	timePause  time.Time
	timePaused time.Duration
}

func (s *Stats) String() string {
	return fmt.Sprintf("%v bytes | %-v | %0.4f MB/s", s.Bytes(), s.Duration(), s.Bandwidth())
}
