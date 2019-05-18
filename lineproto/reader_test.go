package lineproto

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"testing"
)

func TestReader(t *testing.T) {
	var byts []byte
	byts = append(byts, []byte("$ZOn|")...)
	byts = append(byts, []byte{120, 156, 82, 241, 47, 201, 72, 45, 114,
		206, 207, 205, 77, 204, 75, 81, 40, 73, 45,
		46, 169, 1, 4, 0, 0, 255, 255, 69, 30, 7, 66}...)
	byts = append(byts, []byte("$Uncompressed|")...)
	byts = append(byts, []byte("$ZOn2|")...)
	byts = append(byts, []byte{120, 156, 82, 241, 47, 201, 72, 45, 114,
		206, 207, 205, 77, 204, 75, 81, 40, 73, 45,
		46, 169, 1, 4, 0, 0, 255, 255, 69, 30, 7, 66}...)
	byts = append(byts, []byte("$Uncompressed2|")...)
	byts = append(byts, []byte("binary")...)
	byts = append(byts, []byte("$command3|")...)
	byts = append(byts, []byte("2nary")...)
	byts = append(byts, []byte("$command4|")...)

	r := NewReader(bytes.NewReader(byts), '|')

	expect := func(exp string) {
		line, err := r.ReadLine()
		require.NoError(t, err)
		require.Equal(t, exp, string(line))
	}

	expect("$ZOn|")
	err := r.EnableZlib()
	require.NoError(t, err)

	expect("$OtherCommand test|")
	expect("$Uncompressed|")

	expect("$ZOn2|")
	err = r.EnableZlib()
	require.NoError(t, err)

	expect("$OtherCommand test|")
	expect("$Uncompressed2|")

	// test binary reader
	rc, err := r.Binary(6)
	require.NoError(t, err)

	data, err := ioutil.ReadAll(io.LimitReader(rc, 7))
	require.NoError(t, err)
	require.Equal(t, "binary", string(data))

	err = rc.Close()
	require.NoError(t, err)

	expect("$command3|")

	// test binary reader drain
	rc, err = r.Binary(5)
	require.NoError(t, err)

	// partial read
	data, err = ioutil.ReadAll(io.LimitReader(rc, 3))
	require.NoError(t, err)
	require.Equal(t, "2na", string(data))

	err = rc.Close()
	require.NoError(t, err)

	expect("$command4|")
}
