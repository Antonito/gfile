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

func Test_Run(t *testing.T) {
	assert := assert.New(t)

	args := os.Args[0:1]
	err := run(args)
	assert.Nil(err)
}

func Test_Help(t *testing.T) {
	assert := assert.New(t)

	args := os.Args[0:1]
	args = append(args, "help")
	err := run(args)
	assert.Nil(err)
}

func Test_RunReceive(t *testing.T) {
	assert := assert.New(t)

	args := os.Args[0:1]
	args = append(args, "r")
	err := run(args)
	assert.NotNil(err)

	args = os.Args[0:1]
	args = append(args, "receive")
	err = run(args)
	assert.NotNil(err)

	// TODO: Test correct start ?
}

func Test_RunSend(t *testing.T) {
	assert := assert.New(t)

	args := os.Args[0:1]
	args = append(args, "s")
	err := run(args)
	assert.NotNil(err)

	args = os.Args[0:1]
	args = append(args, "send")
	err = run(args)
	assert.NotNil(err)

	// TODO: Test correct start ?
}
