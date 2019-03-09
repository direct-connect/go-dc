package nmdc

import (
	"bytes"
	"errors"
)

func init() {
	RegisterMessage(&ConnectToMe{})
	RegisterMessage(&RevConnectToMe{})
}

type ConnectToMe struct {
	Targ    string
	Address string
	Secure  bool
}

func (m *ConnectToMe) Type() string {
	return "ConnectToMe"
}

func (m *ConnectToMe) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if err := Name(m.Targ).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteByte(' ')
	buf.WriteString(m.Address)
	if m.Secure {
		buf.WriteByte('S')
	}
	return nil
}

func (m *ConnectToMe) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.Index(data, []byte(" "))
	if i < 0 {
		return errors.New("invalid ConnectToMe command")
	}
	var name Name
	if err := name.UnmarshalNMDC(dec, data[:i]); err != nil {
		return err
	}
	addr := data[i+1:]
	m.Targ = string(name)

	if l := len(addr); l != 0 && addr[l-1] == 'S' {
		addr = addr[:l-1]
		m.Secure = true
	}
	m.Address = string(addr)
	return nil
}

type RevConnectToMe struct {
	From, To string
}

func (m *RevConnectToMe) Type() string {
	return "RevConnectToMe"
}

func (m *RevConnectToMe) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if err := Name(m.From).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteString(" ")
	if err := Name(m.To).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	return nil
}

func (m *RevConnectToMe) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.Index(data, []byte(" "))
	if i < 0 {
		return errors.New("invalid RevConnectToMe command")
	}
	var name Name

	if err := name.UnmarshalNMDC(dec, data[:i]); err != nil {
		return err
	}
	m.From = string(name)

	if err := name.UnmarshalNMDC(dec, data[i+1:]); err != nil {
		return err
	}
	m.To = string(name)

	return nil
}
