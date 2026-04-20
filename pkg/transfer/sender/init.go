package sender

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/antonito/gfile/internal/protocol"
	"github.com/antonito/gfile/pkg/transfer"
)

// Initialize creates the PeerConnection and the data channel, then
// emits the local offer SDP.
func (s *Session) Initialize() error {
	if s.initialized {
		return nil
	}

	if err := s.sess.CreateConnection(s.onConnectionStateChange()); err != nil {
		log.Error().Err(err).Msg("")
		return err
	}
	if err := s.createDataChannel(); err != nil {
		log.Error().Err(err).Msg("")
		return err
	}

	offer, err := s.sess.MakeOffer()
	if err != nil {
		log.Error().Err(err).Msg("")
		return err
	}

	transfer.EmitSDP(s.SDPOutput(), transfer.RoleSender, offer)
	transfer.MaybeShowQR(offer, s.disableQR)
	s.initialized = true

	return nil
}

// Start opens the connection and drives the file transfer.
func (s *Session) Start() error {
	defer func() {
		_ = s.sess.Close()
	}()

	if err := s.Initialize(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelRun = cancel
	defer cancel()

	go func() {
		select {
		case <-s.ch.Closed():
			s.stopSend()
			s.close(true)
		case <-ctx.Done():
		}
	}()

	if s.connections > 1 {
		transfer.RunFrames(ctx, s.ch, &senderCtrlHandler{s: s}, protocol.MaxControlReadBufSize, func(err error) {
			log.Warn().Err(err).Msg("sender control OnFrames")
		})
		go s.runMulti(ctx)
	} else {
		go s.run(ctx)
	}

	answer, err := transfer.ReadRemoteSDP(s.SDPInput())
	if err != nil {
		log.Error().Err(err).Msg("")
		return err
	}

	if err := s.sess.AcceptAnswer(answer); err != nil {
		log.Error().Err(err).Msg("")
		return err
	}
	<-s.sess.Done

	return nil
}

func (s *Session) createDataChannel() error {
	label := protocol.PrimaryLabel
	if s.connections > 1 {
		label = protocol.ControlLabel
	}

	ch, err := s.sess.CreateChannel(label)
	if err != nil {
		return err
	}

	s.ch = ch

	return nil
}
