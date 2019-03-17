package main

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_InitLogger(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		level    string
		expected log.Level
	}{
		{
			level:    "",
			expected: log.WarnLevel,
		},
		{
			level:    "InvalidValue",
			expected: log.WarnLevel,
		},
		{
			level:    "TRACE",
			expected: log.TraceLevel,
		},
		{
			level:    "DEBUG",
			expected: log.DebugLevel,
		},
		{
			level:    "INFO",
			expected: log.InfoLevel,
		},
		{
			level:    "WARN",
			expected: log.WarnLevel,
		},
		{
			level:    "PANIC",
			expected: log.PanicLevel,
		},
		{
			level:    "ERROR",
			expected: log.ErrorLevel,
		},
		{
			level:    "FATAL",
			expected: log.FatalLevel,
		},
	}

	for _, cur := range tests {
		if cur.level == "" {
			os.Unsetenv("GFILE_LOG")
		} else {
			os.Setenv("GFILE_LOG", cur.level)
		}
		setupLogger()
		assert.Equal(cur.expected, log.GetLevel())
	}
}
