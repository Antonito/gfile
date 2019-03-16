package stats

// Bytes returns the stored number of bytes
func (s *Stats) Bytes() uint64 {
	return s.nbBytes
}

// AddBytes increase the nbBytes counter
func (s *Stats) AddBytes(c uint64) {
	s.nbBytes += c
}
