package sender

const (
	bufferThreshold = 512 * 1024 // 512kB
)

// Start the connection and the file transfer
func (s *Session) Start() error {
	if err := s.sess.Start(); err != nil {
		return err
	}

	go s.readFile()

	<-s.sess.Done

	return nil
}
