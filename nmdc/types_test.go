package nmdc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEscaping(t *testing.T) {
	var cases = []struct {
		text string
		exp  string
	}{
		{
			text: "text $&|",
			exp:  "text &#36;&amp;&#124;",
		},
	}
	buf := bytes.NewBuffer(nil)
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			buf.Reset()
			err := String(c.text).MarshalNMDC(nil, buf)
			require.NoError(t, err)
			require.Equal(t, c.exp, buf.String())

			got := UnescapeBytes([]byte(c.exp))
			require.Equal(t, c.text, string(got))
		})
	}
}
