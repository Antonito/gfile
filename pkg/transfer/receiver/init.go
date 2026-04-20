package receiver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"

	"github.com/antonito/gfile/internal/protocol"
	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/pkg/transfer"
)

func (s *Session) onConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		log.Info().Msgf("ICE Connection State has changed: %s", connectionState.String())
	}
}

// Initialize creates the PC, registers the data-channel handler, reads the offer, and emits the answer. ctx bounds the OnFrames goroutines.
func (s *Session) Initialize(ctx context.Context) error {
	if s.initialized {
		return nil
	}
	if err := s.sess.CreateConnection(s.onConnectionStateChange()); err != nil {
		log.Error().Err(err).Msg("")
		return err
	}

	s.createDataHandler(ctx)

	offer, err := transfer.ReadRemoteSDP(s.SDPInput())
	if err != nil {
		log.Error().Err(err).Msg("")
		return err
	}

	answer, err := s.sess.AcceptOffer(offer)
	if err != nil {
		log.Error().Err(err).Msg("")
		return err
	}
	transfer.EmitSDP(s.SDPOutput(), transfer.RoleReceiver, answer)
	s.initialized = true

	return nil
}

// Start initializes the connection and runs the transfer. Cancelling ctx unblocks OnFrames.
func (s *Session) Start(ctx context.Context) error {
	defer func() {
		_ = s.sess.Close()
	}()
	if err := s.Initialize(ctx); err != nil {
		return err
	}

	select {
	case <-s.pathReady:
	case <-time.After(30 * time.Second):
		return errors.New("no DataChannel opened within 30s")
	}

	stopProgress := transfer.StartProgressEmitter(ctx, transfer.RoleReceiver, s.sess.NetworkStats)
	defer stopProgress()

	if s.multi != nil {
		return s.multi.waitDone()
	}

	return s.single.waitDone()
}

func (s *Session) createDataHandler(ctx context.Context) {
	s.sess.OnChannel(func(ch *internalSess.Channel) {
		label := ch.Label()
		log.Debug().Msgf("New DataChannel %s", label)

		switch label {
		case protocol.ControlLabel:
			s.multi = newMultiRouter(s.stream, s.path, s.sess.NetworkStats, s.sess.IsLoopbackOnly(), ch, ctx)
			s.startDataHandler(ctx, ch, s.multi, protocol.MaxControlReadBufSize,
				func(err error) {
					log.Error().Err(err).Msg("control OnFrames exited")
				},
				s.multi.failOnEarlyClose,
			)

		case protocol.PrimaryLabel:
			s.single = newSingleHandler(s.stream, s.path, s.sess.NetworkStats)
			s.startDataHandler(ctx, ch, s.single, protocol.MaxDataReadBufSize,
				func(err error) {
					_ = s.single.core.fail(fmt.Errorf("protocol error: %w", err))
				},
				s.single.failOnEarlyClose,
			)

		default:
			log.Warn().Msgf("ignoring DataChannel with unexpected label %q", label)
		}
	})
}

// startDataHandler wires ch to fh: starts stats, signals pathReady, spawns
// the OnFrames loop, and spawns the close watcher. Shared by the control
// and primary label branches.
func (s *Session) startDataHandler(
	ctx context.Context,
	ch *internalSess.Channel,
	fh protocol.FrameHandler,
	readBufSize int,
	onFramesErr func(error),
	onEarlyClose func() error,
) {
	s.sess.NetworkStats.Start()
	s.pathReadyOnce.Do(func() {
		close(s.pathReady)
	})
	transfer.RunFrames(ctx, ch, fh, readBufSize, onFramesErr)
	go s.watchClose(ch, func() {
		_ = onEarlyClose()
	})
}

func (s *Session) watchClose(ch *internalSess.Channel, onClose func()) {
	<-ch.Closed()
	onClose()

	s.sess.NetworkStats.Stop()
}
