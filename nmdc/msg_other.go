package nmdc

import (
	"bytes"
)

func init() {
	RegisterMessage(&ForceMove{})
	RegisterMessage(&Kick{})
	RegisterMessage(&Close{})
	RegisterMessage(&GetTopic{})
	RegisterMessage(&SetTopic{})
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

var _ Message = (*Kick)(nil)

type Kick struct {
	Name
}

func (*Kick) Type() string {
	return "Kick"
}

var _ Message = (*Close)(nil)

type Close struct {
	Name
}

func (*Close) Type() string {
	return "Close"
}

var _ Message = (*GetTopic)(nil)

type GetTopic struct {
	NoArgs
}

func (*GetTopic) Type() string {
	return "GetTopic"
}

var _ Message = (*SetTopic)(nil)

type SetTopic struct {
	String
}

func (*SetTopic) Type() string {
	return "SetTopic"
}
