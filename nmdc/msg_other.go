package nmdc

import (
	"bytes"
	"errors"
)

func init() {
	RegisterMessage(&ForceMove{})
	RegisterMessage(&Kick{})
	RegisterMessage(&Close{})
	RegisterMessage(&GetTopic{})
	RegisterMessage(&SetTopic{})
	// must not register KeepAlive
}

var _ Message = (*KeepAlive)(nil)

type KeepAlive struct{}

func (*KeepAlive) Type() string {
	return "" // special case
}

func (m *KeepAlive) MarshalNMDC(_ *Encoding, buf *bytes.Buffer) error {
	return nil
}

func (m *KeepAlive) UnmarshalNMDC(_ *Encoding, data []byte) error {
	if len(data) != 0 {
		return &ErrProtocolViolation{
			Err: errors.New("nmdc: keep alive message should be empty"),
		}
	}
	return nil
}

type ForceMove struct {
	Address string
}

func (*ForceMove) Type() string {
	return "ForceMove"
}

func (m *ForceMove) MarshalNMDC(_ *Encoding, buf *bytes.Buffer) error {
	buf.WriteString(m.Address)
	return nil
}

func (m *ForceMove) UnmarshalNMDC(_ *Encoding, data []byte) error {
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
