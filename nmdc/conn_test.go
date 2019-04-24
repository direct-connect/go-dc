package nmdc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAddr(t *testing.T) {
	var cases = []struct {
		name string
		addr string
		exp  string
		err  bool
	}{
		{
			name: "only host",
			addr: "localhost",
			exp:  SchemeNMDC + "://localhost",
		},
		{
			name: "only host ipv6",
			addr: "[::]",
			exp:  SchemeNMDC + "://[::]",
		},
		{
			name: "only port",
			addr: ":411",
			err:  true,
		},
		{
			name: "scheme and host",
			addr: "dchub://localhost",
			exp:  SchemeNMDC + "://localhost",
		},
		{
			name: "scheme and host ipv6",
			addr: "dchub://[::]",
			exp:  SchemeNMDC + "://[::]",
		},
		{
			name: "host and port",
			addr: "dchub://localhost:411",
			exp:  SchemeNMDC + "://localhost:411",
		},
		{
			name: "host and port ipv6",
			addr: "dchub://[::]:411",
			exp:  SchemeNMDC + "://[::]:411",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeAddr(c.addr)
			if c.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.exp, got)
		})
	}
}
