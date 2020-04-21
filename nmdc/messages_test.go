package nmdc

import (
	"bytes"
	"testing"

	"github.com/direct-connect/go-dc/tiger"
	"github.com/stretchr/testify/require"
)

type casesMessageEntry struct {
	typ     string
	name    string
	data    string
	expData string
	msg     Message
}

func getTHPointer(s string) *tiger.Hash {
	pointer := tiger.MustParseBase32(s)
	return &pointer
}

func doMessageTestUnmarshal(t *testing.T, cases []casesMessageEntry) {
	for _, raw := range []bool{false, true} {
		ec := DefaultEncoding()
		name := "utf8"
		if raw {
			name = "raw"
			ec = ec.Clone()
			ec.Raw = true
		}
		t.Run(name, func(t *testing.T) {
			for _, c := range cases {
				name := c.typ
				if c.name != "" {
					name += " " + c.name
				}
				t.Run(name, func(t *testing.T) {
					m := NewMessage(c.typ)
					err := m.UnmarshalNMDC(ec, []byte(c.data))
					require.NoError(t, err)
					require.Equal(t, c.msg, m)
				})
			}
		})
	}
}

func doMessageTestMarshal(t *testing.T, cases []casesMessageEntry) {
	for _, raw := range []bool{false, true} {
		ec := DefaultEncoding()
		name := "utf8"
		if raw {
			name = "raw"
			ec = ec.Clone()
			ec.Raw = true
		}
		t.Run(name, func(t *testing.T) {
			for _, c := range cases {
				name := c.typ
				if c.name != "" {
					name += " " + c.name
				}
				t.Run(name, func(t *testing.T) {
					buf := bytes.NewBuffer(nil)
					err := c.msg.MarshalNMDC(ec, buf)
					exp := c.expData
					if exp == "" {
						exp = c.data
					}
					require.NoError(t, err)
					require.Equal(t, exp, buf.String())
				})
			}
		})
	}
}

func doMessageBenchmarkUnmarshal(b *testing.B, cases []casesMessageEntry) {
	for _, raw := range []bool{false, true} {
		ec := DefaultEncoding()
		name := "utf8"
		if raw {
			name = "raw"
			ec = ec.Clone()
			ec.Raw = true
		}
		b.Run(name, func(b *testing.B) {
			for _, c := range cases {
				name := c.typ
				if c.name != "" {
					name += " " + c.name
				}
				b.Run(name, func(b *testing.B) {
					data := []byte(c.data)
					for i := 0; i < b.N; i++ {
						m := NewMessage(c.typ)
						err := m.UnmarshalNMDC(ec, data)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}

func doMessageBenchmarkMarshal(b *testing.B, cases []casesMessageEntry) {
	for _, raw := range []bool{false, true} {
		ec := DefaultEncoding()
		name := "utf8"
		if raw {
			name = "raw"
			ec = ec.Clone()
			ec.Raw = true
		}
		b.Run(name, func(b *testing.B) {
			for _, c := range cases {
				name := c.typ
				if c.name != "" {
					name += " " + c.name
				}
				b.Run(name, func(b *testing.B) {
					m := NewMessage(c.typ)
					err := m.UnmarshalNMDC(ec, []byte(c.data))
					buf := bytes.NewBuffer(nil)
					b.ResetTimer()
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						buf.Reset()
						err = m.MarshalNMDC(ec, buf)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
