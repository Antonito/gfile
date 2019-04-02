package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_OnNewDataChannel(t *testing.T) {
	assert := assert.New(t)

	sess := NewWith(Config{
		Master: false,
	})
	assert.NotNil(sess)

	sess.onNewDataChannel()(nil)

	testID := sess.uploadChannelID()
	sess.onNewDataChannelHelper("", testID, nil)

	testID = sess.uploadChannelID() | sess.downloadChannelID()
	sess.onNewDataChannelHelper("", testID, nil)
}
