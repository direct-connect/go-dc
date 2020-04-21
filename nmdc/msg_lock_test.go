package nmdc

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

var lockCases = []casesMessageEntry{
	{
		typ:  "Lock",
		name: "no pk no ref",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
		},
	},
	{
		typ:  "Lock",
		name: "without Pk",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC Ref=dchub://example.org:411`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
			Ref:  "dchub://example.org:411",
		},
	},
	{
		typ:  "Lock",
		name: "without Ref",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC Pk=DCPLUSPLUS0.777`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
			PK:   "DCPLUSPLUS0.777",
		},
	},
	{
		typ:  "Lock",
		name: "with Ref",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC Pk=DCPLUSPLUS0.777Ref=dchub://example.org:411`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
			PK:   "DCPLUSPLUS0.777",
			Ref:  "dchub://example.org:411",
		},
	},
}

func TestLockUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, lockCases)
}

func TestLockMarshal(t *testing.T) {
	doMessageTestMarshal(t, lockCases)
}

func BenchmarkLockUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, lockCases)
}

func BenchmarkLockMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, lockCases)
}

func TestLockKey(t *testing.T) {
	lock := Lock{
		Lock: "_verlihub",
		PK:   "version0.9.8e-r2",
	}
	key := lock.Key()
	exp, _ := hex.DecodeString("75d1c011b0a010104120d1b1b1c0c03031923171e15010d171")
	require.Equal(t, string(exp), key.Key)
}
