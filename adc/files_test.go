package adc

import (
	"testing"
)

var filesCases = []casesMessageEntry{
	{
		"get file",
		`file TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 124 12352`,
		&GetRequest{
			Type:       "file",
			Path:       "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:      124,
			Bytes:      12352,
			Compressed: false,
		},
	},
	{
		"get file compressed",
		`file TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 124 12352 ZL1`,
		&GetRequest{
			Type:       "file",
			Path:       "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:      124,
			Bytes:      12352,
			Compressed: true,
		},
	},
	{
		"get tthl",
		`tthl TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 124 12352`,
		&GetRequest{
			Type:       "tthl",
			Path:       "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:      124,
			Bytes:      12352,
			Compressed: false,
		},
	},
}

func TestFilesUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, filesCases)
}

func TestFilesMarshal(t *testing.T) {
	doMessageTestMarshal(t, filesCases)
}
