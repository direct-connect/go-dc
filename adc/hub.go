package adc

import (
	"bytes"
	"encoding/base32"

	"github.com/direct-connect/go-dc/tiger"
)

func init() {
	RegisterMessage(SIDAssign{})
	RegisterMessage(UserCommand{})
	RegisterMessage(GetPassword{})
	RegisterMessage(Password{})
	RegisterMessage(Disconnect{})
}

var (
	_ Message     = SIDAssign{}
	_ Marshaler   = SIDAssign{}
	_ Unmarshaler = (*SIDAssign)(nil)
)

type SIDAssign struct {
	SID SID
}

func (SIDAssign) Cmd() MsgType {
	return MsgType{'S', 'I', 'D'}
}

func (m SIDAssign) MarshalADC(buf *bytes.Buffer) error {
	return m.SID.MarshalADC(buf)
}

func (m *SIDAssign) UnmarshalADC(data []byte) error {
	return m.SID.UnmarshalADC(data)
}

var _ Message = UserCommand{}

type Category int

const (
	CategoryHub      = Category(1)
	CategoryUser     = Category(2)
	CategorySearch   = Category(4)
	CategoryFileList = Category(8)
)

type UserCommand struct {
	Path        Path     `adc:"#"`
	Command     string   `adc:"TT"`
	Category    Category `adc:"CT"`
	Remove      int      `adc:"RM"`
	Constrained int      `adc:"CO"`
	Separator   int      `adc:"SP"`
}

func (UserCommand) Cmd() MsgType {
	return MsgType{'C', 'M', 'D'}
}

var base32Enc = base32.StdEncoding.WithPadding(base32.NoPadding)

var (
	_ Message     = GetPassword{}
	_ Marshaler   = GetPassword{}
	_ Unmarshaler = (*GetPassword)(nil)
)

type GetPassword struct {
	Salt []byte
}

func (GetPassword) Cmd() MsgType {
	return MsgType{'G', 'P', 'A'}
}

func (m GetPassword) MarshalADC(buf *bytes.Buffer) error {
	size := base32Enc.EncodedLen(len(m.Salt))
	data := make([]byte, size)
	base32Enc.Encode(data, m.Salt)
	buf.Write(data)
	return nil
}

func (m *GetPassword) UnmarshalADC(data []byte) error {
	size := base32Enc.DecodedLen(len(data))
	m.Salt = make([]byte, size)
	_, err := base32Enc.Decode(m.Salt, data)
	return err
}

var _ Message = Password{}

type Password struct {
	Hash tiger.Hash `adc:"#"`
}

func (Password) Cmd() MsgType {
	return MsgType{'P', 'A', 'S'}
}

var (
	_ Message = Disconnect{}
)

type Disconnect struct {
	ID       SID    `adc:"#"`
	Message  string `adc:"MS"`
	By       SID    `adc:"ID"`
	Duration int    `adc:"TL"`
	Redirect string `adc:"RD"`
	// TODO: "DI"
}

func (Disconnect) Cmd() MsgType {
	return MsgType{'Q', 'U', 'I'}
}
