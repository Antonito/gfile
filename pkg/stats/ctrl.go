package stats

import "time"

// Start stores the "start" timestamp
func (s *Stats) Start() {
	if s.timeStart.IsZero() {
		s.timeStart = time.Now()
	} else if !s.timePause.IsZero() {
		s.timePaused += time.Since(s.timePause)
		// Reset
		s.timePause = time.Time{}
	}
}

// Pause stores an interruption timestamp
func (s *Stats) Pause() {
	if s.timeStart.IsZero() || !s.timeStop.IsZero() {
		// Can't stop if not started, or if stopped
		return
	}
	if s.timePause.IsZero() {
		s.timePause = time.Now()
	}
}

// Stop stores the "stop" timestamp
func (s *Stats) Stop() {
	if s.timeStart.IsZero() {
		// Can't stop if not started
		return
	}
	if s.timeStop.IsZero() {
		s.timeStop = time.Now()
	}
}
