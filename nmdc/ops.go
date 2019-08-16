package nmdc

import (
	"bytes"
)

func init() {
	RegisterMessage(&ForceMove{})
}

type ForceMove struct {
	Address string
}

func (*ForceMove) Type() string {
	return "ForceMove"
}

func (m *ForceMove) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	buf.WriteString(m.Address)
	return nil
}

func (m *ForceMove) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	m.Address = string(data)
	return nil
}
