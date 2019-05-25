package nmdc

import (
	"errors"
	"testing"
)

var errorsCases = []casesMessageEntry{
	{
		typ:  "Error",
		data: `message`,
		msg: &Error{
			Err: errors.New("message"),
		},
	},
}

func TestErrorsUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, errorsCases)
}

func TestErrorsMarshal(t *testing.T) {
	doMessageTestMarshal(t, errorsCases)
}

func BenchmarkErrorsUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, errorsCases)
}

func BenchmarkErrorsMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, errorsCases)
}
