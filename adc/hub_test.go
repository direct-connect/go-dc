package adc

import (
	"testing"

	"github.com/direct-connect/go-dc/tiger"
)

var hubCases = []casesMessageEntry{
	{
		"user command",
		`ADCH++/Hub\smanagement/Reload\sscripts TTHMSG\s+reload\n CT3`,
		&UserCommand{
			Path:     Path{"ADCH++", "Hub management", "Reload scripts"},
			Command:  "HMSG +reload\n",
			Category: CategoryHub | CategoryUser,
		},
	},
	{
		"get password",
		`AAAQEAYEAUDAOCAJAAAQEAYCAMCAKBQHBAEQAAI`,
		&GetPassword{Salt: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1}},
	},
	{
		"password",
		`ABZCJESSJKVMIL2BDERHSJ7RF5IYI6ZX2QAOQGI`,
		&Password{Hash: tiger.HashBytes([]byte("qwerty"))},
	},
}

func TestHubUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, hubCases)
}

func TestHubMarshal(t *testing.T) {
	doMessageTestMarshal(t, hubCases)
}
