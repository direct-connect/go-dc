package nmdc

import (
	"testing"
)

var filesCases = []casesMessageEntry{
	{
		typ:  "ADCGET",
		data: `file files.xml.bz2 0 -1`,
		msg: &ADCGet{
			ContentType: "file",
			Identifier:  "files.xml.bz2",
			Start:       0,
			Length:      -1,
		},
	},
	{
		typ:  "ADCGET",
		data: `file TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 123523 65`,
		msg: &ADCGet{
			ContentType: "file",
			Identifier:  "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:       123523,
			Length:      65,
		},
	},
	{
		typ:  "ADCGET",
		data: `tthl TTH/PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I 6342 7323`,
		msg: &ADCGet{
			ContentType: "tthl",
			Identifier:  "TTH/PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I",
			Start:       6342,
			Length:      7323,
		},
	},
	{
		typ:  "ADCGET",
		data: `file TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 123523 65 ZL1`,
		msg: &ADCGet{
			ContentType: "file",
			Identifier:  "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:       123523,
			Length:      65,
			Compressed:  true,
		},
	},
	{
		typ:  "ADCGET",
		data: `file TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 123523 65 ZL1 DB15`,
		msg: &ADCGet{
			ContentType: "file",
			Identifier:  "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:       123523,
			Length:      65,
			Compressed:  true,
			DownloadedBytes: func() *uint64 {
				ret := new(uint64)
				*ret = 15
				return ret
			}(),
		},
	},
	{
		typ:  "ADCGET",
		data: `file TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 123523 65 DB15`,
		msg: &ADCGet{
			ContentType: "file",
			Identifier:  "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:       123523,
			Length:      65,
			Compressed:  false,
			DownloadedBytes: func() *uint64 {
				ret := new(uint64)
				*ret = 15
				return ret
			}(),
		},
	},
	{
		typ:  "ADCSND",
		data: `file TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI 123523 65`,
		msg: &ADCSnd{
			ContentType: "file",
			Identifier:  "TTH/BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI",
			Start:       123523,
			Length:      65,
		},
	},
	{
		typ:  "ADCSND",
		data: `tthl TTH/PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I 6342 7323`,
		msg: &ADCSnd{
			ContentType: "tthl",
			Identifier:  "TTH/PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I",
			Start:       6342,
			Length:      7323,
		},
	},
	{
		typ:  "Direction",
		data: `Download 12343`,
		msg: &Direction{
			Upload: false,
			Number: 12343,
		},
	},
	{
		typ:  "Direction",
		data: `Upload 6533`,
		msg: &Direction{
			Upload: true,
			Number: 6533,
		},
	},
}

func TestFilesUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, filesCases)
}

func TestFilesMarshal(t *testing.T) {
	doMessageTestMarshal(t, filesCases)
}

func BenchmarkFilesUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, filesCases)
}

func BenchmarkFilesMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, filesCases)
}
