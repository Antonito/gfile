package receiver

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	asrt := assert.New(t)
	file, err := os.CreateTemp(t.TempDir(), "receiver_test_*")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	sess := New(file)

	asrt.NotNil(sess)
	asrt.Equal(file, sess.stream)
}
