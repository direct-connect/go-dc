package lineproto

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReader(t *testing.T) {
	var byts []byte
	byts = append(byts, []byte("$ZOn|")...)
	byts = append(byts, []byte{120, 156, 82, 241, 47, 201, 72, 45, 114,
		206, 207, 205, 77, 204, 75, 81, 40, 73, 45,
		46, 169, 1, 4, 0, 0, 255, 255, 69, 30, 7, 66}...)
	byts = append(byts, []byte("$Uncompressed|")...)

	r := NewReader(bytes.NewReader(byts), '|')

	expect := func(exp string) {
		line, err := r.ReadLine()
		require.NoError(t, err)
		require.Equal(t, exp, string(line))
	}

	expect("$ZOn|")

	err := r.ActivateZlib()
	require.NoError(t, err)

	expect("$OtherCommand test|")
	expect("$Uncompressed|")
}
