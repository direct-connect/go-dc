package adc

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/direct-connect/go-dc/adc/types"
	"github.com/stretchr/testify/require"
)

func TestSIDMarshal(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	err := types.SIDFromString("AAAA").MarshalADC(buf)
	require.NoError(t, err)
	require.Equal(t, "AAAA", buf.String())

	buf = bytes.NewBuffer(nil)
	err = types.SIDFromInt(2).MarshalADC(buf)
	require.NoError(t, err)
	require.Equal(t, "AAAC", buf.String())

	buf = bytes.NewBuffer(nil)
	err = types.SIDFromInt(34).MarshalADC(buf)
	require.NoError(t, err)
	require.Equal(t, "AABC", buf.String())
}

func sidp(s string) *types.SID {
	v := types.SIDFromString(s)
	return &v
}

type casesMessageEntry struct {
	name string
	data string
	msg  Message
}

func doMessageTestUnmarshal(t *testing.T, cases []casesMessageEntry) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			targ := reflect.New(reflect.TypeOf(c.msg).Elem()).Interface()
			err := Unmarshal([]byte(c.data), targ)
			require.NoError(t, err)
			require.Equal(t, c.msg, targ)
		})
	}
}

func doMessageTestMarshal(t *testing.T, cases []casesMessageEntry) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			err := Marshal(buf, c.msg)
			require.NoError(t, err)
			require.Equal(t, c.data, buf.String())
		})
	}
}
