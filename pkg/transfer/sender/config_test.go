package sender

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name             string
		compressionLevel int
		connections      int
		wantErr          bool
		errContains      string
	}{
		{name: "valid mid-range", compressionLevel: 3, connections: 2, wantErr: false},
		{name: "compression 0 ok", compressionLevel: 0, connections: 1, wantErr: false},
		{name: "compression 22 ok", compressionLevel: 22, connections: 1, wantErr: false},
		{name: "compression -1 err", compressionLevel: -1, connections: 1, wantErr: true, errContains: "compression level"},
		{name: "compression 23 err", compressionLevel: 23, connections: 1, wantErr: true, errContains: "compression level"},
		{name: "connections 1 ok", compressionLevel: 0, connections: 1, wantErr: false},
		{name: "connections 16 ok", compressionLevel: 0, connections: 16, wantErr: false},
		{name: "connections 0 err", compressionLevel: 0, connections: 0, wantErr: true, errContains: "connections"},
		{name: "connections 17 err", compressionLevel: 0, connections: 17, wantErr: true, errContains: "connections"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{CompressionLevel: tt.compressionLevel, Connections: tt.connections}
			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
