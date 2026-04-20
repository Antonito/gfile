package sender

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/klauspost/compress/zstd"

	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/internal/stats"
	"github.com/antonito/gfile/pkg/transfer"
)

// Session is a sender session.
type Session struct {
	transfer.SessionBase
	sess        internalSess.Session
	stream      io.ReadSeeker
	initialized bool
	disableQR   bool

	ch        *internalSess.Channel
	cancelRun context.CancelFunc

	// populated by the control-channel OnFrames handler
	answerCh chan peerAnswer

	encoder     *zstd.Encoder
	zstdLevel   int
	connections int
	dataPeers   []*dataPeer

	closeOnce sync.Once

	readingStats *stats.Stats
}

func newSession(
	sess internalSess.Session,
	file io.ReadSeeker,
	level int,
	connections int,
	ioCfg transfer.IOConfig,
) (*Session, error) {
	session := &Session{
		SessionBase:  transfer.NewSessionBase(sess.NetworkStats, ioCfg),
		sess:         sess,
		stream:       file,
		connections:  connections,
		zstdLevel:    level,
		disableQR:    ioCfg.DisableQR,
		readingStats: stats.New(),
	}

	if connections > 1 {
		session.answerCh = make(chan peerAnswer, connections)
	}
	// Single-PC mode reuses one session-wide encoder. Multi-PC builds
	// per-worker encoders in multiWorker so encoding parallelizes; the
	// session-level encoder would just sit unused.
	if level > 0 && connections == 1 {
		enc, err := transfer.NewDataEncoder(level)
		if err != nil {
			return nil, fmt.Errorf("zstd encoder: %w", err)
		}
		session.encoder = enc
	}

	return session, nil
}

// New returns a single-PC sender with stdin/stdout SDP and default settings.
func New(file io.ReadSeeker) *Session {
	session, _ := newSession(internalSess.New(internalSess.Config{}), file, 1, 1, transfer.IOConfig{})
	return session
}

// Config holds sender-side configuration.
type Config struct {
	// IOConfig carries SDP I/O, STUN, QR, and loopback settings.
	transfer.IOConfig
	// Stream is the file/reader to transmit. Must be seekable (pre-hash rewinds).
	Stream io.ReadSeeker
	// CompressionLevel is the zstd level (0 disables; 1..22 map to klauspost buckets).
	CompressionLevel int
	// Connections is the number of parallel data PeerConnections (1..16).
	Connections int
}

// Validate checks Config fields. NewWith does not re-validate.
func (c Config) Validate() error {
	if c.CompressionLevel < 0 || c.CompressionLevel > 22 {
		return fmt.Errorf("compression level must be in [0, 22] (got %d)", c.CompressionLevel)
	}
	if c.Connections < 1 || c.Connections > 16 {
		return fmt.Errorf("connections must be in [1, 16] (got %d)", c.Connections)
	}

	return nil
}

// NewWith returns a sender configured from cfg. Call Config.Validate() first.
func NewWith(cfg Config) (*Session, error) {
	sess := internalSess.New(transfer.BuildInternalConfig(cfg.IOConfig))
	return newSession(sess, cfg.Stream, cfg.CompressionLevel, cfg.Connections, cfg.IOConfig)
}

var _ io.ReadSeeker = (*os.File)(nil)
