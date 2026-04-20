package debug

import (
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// StartPprof starts a pprof HTTP server on the address read from envVar;
// no-op if the variable is unset.
//
// A listener failure is fatal
func StartPprof(envVar string) {
	addr := os.Getenv(envVar)
	if addr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Str("env", envVar).Str("addr", addr).Msg("pprof listener failed")
		}
	}()

	log.Info().Str("env", envVar).Str("addr", addr).Msg("pprof listening")
}
