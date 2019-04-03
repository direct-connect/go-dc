package nmdc

import (
	"bytes"
	"errors"
	"fmt"
)

func init() {
	RegisterMessage(&ConnectToMe{})
	RegisterMessage(&RevConnectToMe{})
}

type CTMKind int

const (
	CTMActive = CTMKind(iota)
	CTMPassiveReq
	CTMPassiveResp
)

type ConnectToMe struct {
	Targ    string
	Src     string
	Address string
	Kind    CTMKind
	Secure  bool
}

func (m *ConnectToMe) Type() string {
	return "ConnectToMe"
}

func (m *ConnectToMe) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if m.Targ == "" {
		return fmt.Errorf("ConnectToMe target should be set")
	}
	if err := Name(m.Targ).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteByte(' ')
	buf.WriteString(m.Address)
	switch m.Kind {
	case CTMActive:
	// nothing
	case CTMPassiveReq:
		buf.WriteByte('N')
	case CTMPassiveResp:
		buf.WriteByte('R')
	default:
		return fmt.Errorf("unknown ConnectToMe kind: %v", m.Kind)
	}
	if m.Secure {
		buf.WriteByte('S')
	}
	if m.Kind != CTMPassiveReq {
		if m.Src != "" {
			return fmt.Errorf("only passive ConnectToMe requests should have a source")
		}
		return nil
	}
	if m.Src == "" {
		return fmt.Errorf("passive ConnectToMe requests should have a source")
	}
	buf.WriteByte(' ')
	if err := Name(m.Src).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	return nil
}

func (m *ConnectToMe) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("invalid ConnectToMe command")
	}
	var name Name
	if err := name.UnmarshalNMDC(dec, data[:i]); err != nil {
		return err
	}
	data = data[i+1:]
	m.Targ = string(name)

	if i := bytes.IndexByte(data, ' '); i >= 0 {
		name = ""
		if err := name.UnmarshalNMDC(dec, data[i+1:]); err != nil {
			return err
		}
		data = data[:i]
		m.Src = string(name)
	}
	addr := data

	if l := len(addr); l != 0 && addr[l-1] == 'S' {
		addr = addr[:l-1]
		m.Secure = true
	}
	m.Kind = CTMActive
	if l := len(addr); l != 0 {
		switch addr[l-1] {
		case 'N':
			m.Kind = CTMPassiveReq
			addr = addr[:l-1]
		case 'R':
			m.Kind = CTMPassiveResp
			addr = addr[:l-1]
		}
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
	i := bytes.IndexByte(data, ' ')
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
