package receiver

import (
	"os"
	"sync"

	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/pkg/transfer"
)

// Session is a receiver session.
type Session struct {
	transfer.SessionBase
	sess        internalSess.Session
	stream      *os.File
	initialized bool
	path        string // filesystem path for cleanup on failure; "" disables

	single        *singleHandler
	multi         *multiRouter  // nil in single-PC mode
	pathReady     chan struct{} // closed by OnChannel once single or multi is set
	pathReadyOnce sync.Once
}

func newSession(sess internalSess.Session, file *os.File, path string, io transfer.IOConfig) *Session {
	return &Session{
		SessionBase: transfer.NewSessionBase(sess.NetworkStats, io),
		sess:        sess,
		stream:      file,
		path:        path,
		initialized: false,
		pathReady:   make(chan struct{}),
	}
}

// New creates a receiver writing to file. Cancelling the Start ctx unblocks OnFrames.
func New(file *os.File) *Session {
	sess := internalSess.NewReceiver(internalSess.Config{})
	return newSession(sess, file, "", transfer.IOConfig{})
}

// Config holds receiver-side configuration.
type Config struct {
	// IOConfig carries SDP I/O, STUN, QR, and loopback settings.
	transfer.IOConfig
	// Stream is the file the receiver writes into.
	Stream *os.File
	// Path is an optional filesystem path used for cleanup-on-failure.
	// Empty disables removal.
	Path string
}

// NewWith creates a receiver configured from cfg.
func NewWith(cfg Config) *Session {
	sess := internalSess.NewReceiver(transfer.BuildInternalConfig(cfg.IOConfig))
	return newSession(sess, cfg.Stream, cfg.Path, cfg.IOConfig)
}
