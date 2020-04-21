package nmdc

import (
	"testing"
)

var hubCases = []casesMessageEntry{
	{
		typ:  "FailOver",
		data: `example.com,example.org:5555,adc://example.net:6666`,
		msg: &FailOver{
			Host: []string{
				"example.com",
				"example.org:5555",
				"adc://example.net:6666",
			},
		},
	},
}

func TestHubUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, hubCases)
}

func TestHubMarshal(t *testing.T) {
	doMessageTestMarshal(t, hubCases)
}

func BenchmarkHubUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, hubCases)
}

func BenchmarkHubMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, hubCases)
}
