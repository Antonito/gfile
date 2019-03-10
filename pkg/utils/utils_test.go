package utils

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadStdin(t *testing.T) {
	assert := assert.New(t)
	stream := &bytes.Buffer{}

	_, err := stream.WriteString("Hello\n")
	assert.Nil(err)

	str, err := mustRead(stream)
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
		// Empty input
		{
			input:     nil,
			shouldErr: false,
			expected:  "bnVsbA==",
		},
		// Not JSON
		{
			input:     "ThisTestIsNotInB64",
			shouldErr: false,
			expected:  base64.StdEncoding.EncodeToString([]byte("\"ThisTestIsNotInB64\"")),
		},
		// JSON
		{
			input: struct {
				Name string `json:"name"`
			}{
				Name: "TestJson",
			},
			shouldErr: false,
			expected:  "eyJuYW1lIjoiVGVzdEpzb24ifQ==",
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
			input:     "eyJ0eXBlIjoiYW5zd2VyIiwic2RwIjoidj0wXHJcbm89LSA4MzU4MzUxNDggMTU1MjE4NzMxMyBJTiBJUDQgMC4wLjAuMFxyXG5zPS1cclxudD0wIDBcclxuYT1maW5nZXJwcmludDpzaGEtMjU2IDcwOjZGOkREOjE3OkM1OjkxOkU2OkQ1OjJGOjI0OjkwOjNDOjk5OkQzOkJEOkM0OkFGOkNBOjUxOkU0OjEyOjQ3OjkxOjcwOjU3OkQxOjY2OjRDOjRCOjdEOjI5OjUwXHJcbmE9Z3JvdXA6QlVORExFIGF1ZGlvIHZpZGVvIGRhdGFcclxubT1hdWRpbyA5IFVEUC9UTFMvUlRQL1NBVlBGIDExMSA5XHJcbmM9SU4gSVA0IDAuMC4wLjBcclxuYT1zZXR1cDphY3RpdmVcclxuYT1taWQ6YXVkaW9cclxuYT1pY2UtdWZyYWc6ZVlTUFd1dkhPbG5TVFBIWVxyXG5hPWljZS1wd2Q6WFVaVE5wT0ZRTFlyQ2RPakFUcEp1b3JHS050UlVMSXZcclxuYT1ydGNwLW11eFxyXG5hPXJ0Y3AtcnNpemVcclxuYT1ydHBtYXA6MTExIG9wdXMvNDgwMDAvMlxyXG5hPWZtdHA6MTExIG1pbnB0aW1lPTEwO3VzZWluYmFuZGZlYz0xXHJcbmE9cnRwbWFwOjkgRzcyMi84MDAwXHJcbmE9aW5hY3RpdmVcclxuYT1jYW5kaWRhdGU6Zm91bmRhdGlvbiAxIHVkcCAzNzc2IDE5Mi4xNjguMC4xODkgNjM2OTQgdHlwIGhvc3QgZ2VuZXJhdGlvbiAwXHJcbmE9Y2FuZGlkYXRlOmZvdW5kYXRpb24gMiB1ZHAgMzc3NiAxOTIuMTY4LjAuMTg5IDYzNjk0IHR5cCBob3N0IGdlbmVyYXRpb24gMFxyXG5hPWNhbmRpZGF0ZTpmb3VuZGF0aW9uIDEgdWRwIDMxMDAgNDcuMTQ3LjEyOS4zMiA1MTEzNiB0eXAgc3JmbHggcmFkZHIgMTkyLjE2OC4wLjE4OSBycG9ydCA1MTEzNiBnZW5lcmF0aW9uIDBcclxuYT1jYW5kaWRhdGU6Zm91bmRhdGlvbiAyIHVkcCAzMTAwIDQ3LjE0Ny4xMjkuMzIgNTExMzYgdHlwIHNyZmx4IHJhZGRyIDE5Mi4xNjguMC4xODkgcnBvcnQgNTExMzYgZ2VuZXJhdGlvbiAwXHJcbmE9ZW5kLW9mLWNhbmRpZGF0ZXNcclxubT12aWRlbyA5IFVEUC9UTFMvUlRQL1NBVlBGIDk2IDEwMCA5OFxyXG5jPUlOIElQNCAwLjAuMC4wXHJcbmE9c2V0dXA6YWN0aXZlXHJcbmE9bWlkOnZpZGVvXHJcbmE9aWNlLXVmcmFnOmVZU1BXdXZIT2xuU1RQSFlcclxuYT1pY2UtcHdkOlhVWlROcE9GUUxZckNkT2pBVHBKdW9yR0tOdFJVTEl2XHJcbmE9cnRjcC1tdXhcclxuYT1ydGNwLXJzaXplXHJcbmE9cnRwbWFwOjk2IFZQOC85MDAwMFxyXG5hPXJ0cG1hcDoxMDAgSDI2NC85MDAwMFxyXG5hPWZtdHA6MTAwIGxldmVsLWFzeW1tZXRyeS1hbGxvd2VkPTE7cGFja2V0aXphdGlvbi1tb2RlPTE7cHJvZmlsZS1sZXZlbC1pZD00MjAwMWZcclxuYT1ydHBtYXA6OTggVlA5LzkwMDAwXHJcbmE9aW5hY3RpdmVcclxuYT1jYW5kaWRhdGU6Zm91bmRhdGlvbiAxIHVkcCAzNzc2IDE5Mi4xNjguMC4xODkgNjM2OTQgdHlwIGhvc3QgZ2VuZXJhdGlvbiAwXHJcbmE9Y2FuZGlkYXRlOmZvdW5kYXRpb24gMiB1ZHAgMzc3NiAxOTIuMTY4LjAuMTg5IDYzNjk0IHR5cCBob3N0IGdlbmVyYXRpb24gMFxyXG5hPWNhbmRpZGF0ZTpmb3VuZGF0aW9uIDEgdWRwIDMxMDAgNDcuMTQ3LjEyOS4zMiA1MTEzNiB0eXAgc3JmbHggcmFkZHIgMTkyLjE2OC4wLjE4OSBycG9ydCA1MTEzNiBnZW5lcmF0aW9uIDBcclxuYT1jYW5kaWRhdGU6Zm91bmRhdGlvbiAyIHVkcCAzMTAwIDQ3LjE0Ny4xMjkuMzIgNTExMzYgdHlwIHNyZmx4IHJhZGRyIDE5Mi4xNjguMC4xODkgcnBvcnQgNTExMzYgZ2VuZXJhdGlvbiAwXHJcbmE9ZW5kLW9mLWNhbmRpZGF0ZXNcclxubT1hcHBsaWNhdGlvbiA5IERUTFMvU0NUUCA1MDAwXHJcbmM9SU4gSVA0IDAuMC4wLjBcclxuYT1zZXR1cDphY3RpdmVcclxuYT1taWQ6ZGF0YVxyXG5hPXNlbmRyZWN2XHJcbmE9c2N0cG1hcDo1MDAwIHdlYnJ0Yy1kYXRhY2hhbm5lbCAxMDI0XHJcbmE9aWNlLXVmcmFnOmVZU1BXdXZIT2xuU1RQSFlcclxuYT1pY2UtcHdkOlhVWlROcE9GUUxZckNkT2pBVHBKdW9yR0tOdFJVTEl2XHJcbmE9Y2FuZGlkYXRlOmZvdW5kYXRpb24gMSB1ZHAgMzc3NiAxOTIuMTY4LjAuMTg5IDYzNjk0IHR5cCBob3N0IGdlbmVyYXRpb24gMFxyXG5hPWNhbmRpZGF0ZTpmb3VuZGF0aW9uIDIgdWRwIDM3NzYgMTkyLjE2OC4wLjE4OSA2MzY5NCB0eXAgaG9zdCBnZW5lcmF0aW9uIDBcclxuYT1jYW5kaWRhdGU6Zm91bmRhdGlvbiAxIHVkcCAzMTAwIDQ3LjE0Ny4xMjkuMzIgNTExMzYgdHlwIHNyZmx4IHJhZGRyIDE5Mi4xNjguMC4xODkgcnBvcnQgNTExMzYgZ2VuZXJhdGlvbiAwXHJcbmE9Y2FuZGlkYXRlOmZvdW5kYXRpb24gMiB1ZHAgMzEwMCA0Ny4xNDcuMTI5LjMyIDUxMTM2IHR5cCBzcmZseCByYWRkciAxOTIuMTY4LjAuMTg5IHJwb3J0IDUxMTM2IGdlbmVyYXRpb24gMFxyXG5hPWVuZC1vZi1jYW5kaWRhdGVzXHJcbiJ9",
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
	assert.Equal("eyJuYW1lIjoiVGVzdEpzb24ifQ==", encoded)

	var obj struct {
		Name string `json:"name"`
	}
	err = Decode(encoded, &obj)
	assert.Nil(err)
	assert.Equal(input, obj)
}
