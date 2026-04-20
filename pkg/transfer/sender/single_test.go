package sender

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, data []byte) *os.File {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "in.bin")
	require.NoError(t, os.WriteFile(path, data, 0o600), "seed")
	file, err := os.Open(path)
	require.NoError(t, err, "open")
	t.Cleanup(func() { file.Close() })
	return file
}

func TestPreHashMatchesFile(t *testing.T) {
	asrt := assert.New(t)
	cases := [][]byte{
		nil,
		{0x00},
		bytes.Repeat([]byte{0xab}, 16*1024),
		bytes.Repeat([]byte{0xcd}, 1024*1024+7),
	}
	for ndx, data := range cases {
		file := writeTempFile(t, data)
		got, size, err := preHash(file)
		require.NoErrorf(t, err, "case %d: preHash", ndx)
		asrt.Equalf(uint64(len(data)), size, "case %d: size", ndx)
		want := sha256.Sum256(data)
		asrt.Equalf(want, got, "case %d: hash", ndx)
		pos, _ := file.Seek(0, 1)
		asrt.Equalf(int64(0), pos, "case %d: file position (should be rewound)", ndx)
	}
}
