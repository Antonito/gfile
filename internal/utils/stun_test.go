package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_STUN_Arg(t *testing.T) {
	asrt := assert.New(t)

	tests := []struct {
		input string
		err   error
	}{
		{
			input: "",
			err:   fmt.Errorf("invalid stun address"),
		},
		{
			input: "test",
			err:   fmt.Errorf("invalid stun address"),
		},
		{
			input: "stun:lol:lol",
			err:   fmt.Errorf("invalid stun address"),
		},
		{
			input: "test:wtf",
			err:   fmt.Errorf("invalid port 0"),
		},
		{
			input: "test:-2",
			err:   fmt.Errorf("invalid port -2"),
		},
		{
			// 0xffff + 1
			input: "test:65536",
			err:   fmt.Errorf("invalid port 65536"),
		},
		{
			input: "test:5432",
			err:   nil,
		},
		{
			input: "stun.l.google.com:19302",
			err:   nil,
		},
	}

	for _, cur := range tests {
		err := ParseSTUN(cur.input)
		asrt.Equal(cur.err, err)
	}
}
