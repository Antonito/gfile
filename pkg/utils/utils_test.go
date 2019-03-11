package utils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadStream(t *testing.T) {
	assert := assert.New(t)
	stream := &bytes.Buffer{}

	_, err := stream.WriteString("Hello\n")
	assert.Nil(err)

	str, err := MustReadStream(stream)
	assert.Equal("Hello", str)
	assert.Nil(err)
}

func Test_Encode(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		input     interface{}
		shouldErr bool
		expected  string
	}{
		// Invalid object
		{
			input:     make(chan int),
			shouldErr: true,
		},
		// Empty input
		{
			input:     nil,
			shouldErr: false,
			expected:  "H4sIAAAAAAAC/8orzckBAAAA//8BAAD//0/8yyUEAAAA",
		},
		// Not JSON
		{
			input:     "ThisTestIsNotInB64",
			shouldErr: false,
			expected:  "H4sIAAAAAAAC/1IKycgsDkktLvEs9ssv8cxzMjNRAgAAAP//AQAA//8+sWiWFAAAAA==",
		},
		// JSON
		{
			input: struct {
				Name string `json:"name"`
			}{
				Name: "TestJson",
			},
			shouldErr: false,
			expected:  "H4sIAAAAAAAC/6pWykvMTVWyUgpJLS7xKs7PU6oFAAAA//8BAAD//3cqgZQTAAAA",
		},
	}

	for _, cur := range tests {
		res, err := Encode(cur.input)

		if cur.shouldErr {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
			assert.Equal(cur.expected, res)
		}
	}
}

func Test_Decode(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		input     string
		shouldErr bool
	}{
		// Empty string
		{
			input:     "",
			shouldErr: true,
		},
		// Not base64
		{
			input:     "ThisTestIsNotInB64",
			shouldErr: true,
		},
		// Not base64 JSON
		{
			input:     "aGVsbG8gd29ybGQ=",
			shouldErr: true,
		},
		// Base64 JSON
		{
			input:     "H4sIAAAAAAAC/+xVTY/bNhD9KwOdK5ukqK8JdFh77XabNPXGXqcJcmFEystGogiKsuMU/e+FLG/qtEWBBRo0h4UgCTPz+OaRegP9FvijVQEGwnQH5YLvgk7aAIN9Qd65d6YtQojzmCRpmmZA45ixhFNO4eYl3Kw4kMnpGqBdEQ4vXxA4xaKotNkpZ502Hrt7EbI4gYjiLMKE4oIhYZgskWXIyfCcLZFdY5ZgxnEeYTRDHmGeYsyQ57icIWdI5ji/wnmClCBLMUowjjBLcXGFEUHOx8Y71/YWZ3cvr18sQPRSt7DXUrUghRcDpCnGbA5316vp5sV6+mqzmq6vtqslUEohH0Bl8fdNiqJTvrcoSq/3asw0WuKJbgx1qcK+cmKH5avl6zflzeGu8z9vd4df/qzbg8TXerH7cfnrQj9/u/a36+2b55ubulosDx9u5eb2p7cj2vnShk3/8SJynf6kHmLbCIuD5Nb23ZRnhJApOx9/48dSo431ulEFJc/6TmnzXhhZqbKgX7Dk8H3K2HSgOCs1l9sshZFaCq+wansjhdetAQq9tBClaQI0ZxOaZBMyoVkOcRQnCfijhfu287BTRrlxCfkXOvbf0p3VUUKApxM63CyfRAxinhJ6outcVX8EJ6R0f2npbOv8Gfk4+V+jnzIybKvwc9tutPFo63+ycZ7AoCPPHmvlE+X/ZuU8ge0qm+bkswsfPE4I/MASflkaHU4I1Gqv6lB0x6ZR3h1DUdftQcmCPrOi/KC8/nQ6zLBppRqSrq10rcJxmZYFZ4TQ6kshGWxX+WW3p3H45sdBWFvrckTmcD1MxHq+WUF8/oiPmYOHf8VQN9Kpcn+OytEgAycc1Hvny3DAlvfCGFUDJYx/jfF5Mtw3Z7jg9z8AAAD//wEAAP//RjpVQj8JAAA=",
			shouldErr: false,
		},
	}

	var obj interface{}
	for _, cur := range tests {
		err := Decode(cur.input, &obj)

		if cur.shouldErr {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
	}
}

func Test_EncodeDecode(t *testing.T) {
	assert := assert.New(t)

	input := struct {
		Name string `json:"name"`
	}{
		Name: "TestJson",
	}

	encoded, err := Encode(input)
	assert.Nil(err)
	assert.Equal("H4sIAAAAAAAC/6pWykvMTVWyUgpJLS7xKs7PU6oFAAAA//8BAAD//3cqgZQTAAAA", encoded)

	var obj struct {
		Name string `json:"name"`
	}
	err = Decode(encoded, &obj)
	assert.Nil(err)
	assert.Equal(input, obj)
}
