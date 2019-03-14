package lineproto

import (
	"bytes"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	var byts []byte
	byts = append(byts, []byte("$ZOn|")...)
	byts = append(byts, []byte{120, 156, 82, 241, 47, 201, 72, 45, 114,
		206, 207, 205, 77, 204, 75, 81, 40, 73, 45,
		46, 169, 1, 4, 0, 0, 255, 255, 69, 30, 7, 66}...)
	byts = append(byts, []byte("$Uncompressed|")...)

	r := NewReader(bytes.NewReader(byts), '|')

	l1Expected := []byte("$ZOn|")
	l1, err := r.ReadLine()
	require.NoError(t, err)
	require.Equal(t, l1, l1Expected)

	r.ActivateZlib()

	l2Expected := []byte("$OtherCommand test|")
	l2, err := r.ReadLine()
	require.NoError(t, err)
	require.Equal(t, l2, l2Expected)

	l3Expected := []byte("$Uncompressed|")
	l3, err := r.ReadLine()
	require.NoError(t, err)
	require.Equal(t, l3, l3Expected)
}
