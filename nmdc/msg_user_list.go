package nmdc

import "bytes"

func init() {
	RegisterMessage(&GetNickList{})
	RegisterMessage(&OpList{})
	RegisterMessage(&BotList{})
	RegisterMessage(&UserIP{})
}

// GetNickList is sent by the client to the hub to retrieve a list of online users.
//
// http://nmdc.sourceforge.net/NMDC.html#_getnicklist
type GetNickList struct {
	NoArgs
}

func (*GetNickList) Type() string {
	return "GetNickList"
}

// OpList is a list of hub operators.
//
// http://nmdc.sourceforge.net/NMDC.html#_oplist
type OpList struct {
	Names
}

func (*OpList) Type() string {
	return "OpList"
}

// BotList is a list of bots on the hub. Requires 'BotList' extension.
//
// http://nmdc.sourceforge.net/NMDC.html#_botlist
type BotList struct {
	Names
}

func (*BotList) Type() string {
	return "BotList"
}

// Names is a list of user names separated by '$$'.
//
// See OpList and BotList.
type Names []string

func (m Names) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if len(m) == 0 {
		buf.WriteString("$$")
		return nil
	}
	for _, name := range m {
		if err := Name(name).MarshalNMDC(enc, buf); err != nil {
			return err
		}
		buf.WriteString("$$")
	}
	return nil
}

func (m *Names) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	data = bytes.TrimSuffix(data, []byte("$$"))

	if len(data) == 0 {
		*m = nil
		return nil
	}

	sub := bytes.Split(data, []byte("$$"))
	list := make([]string, 0, len(sub))
	for _, b := range sub {
		var name Name
		if err := name.UnmarshalNMDC(dec, b); err != nil {
			return err
		}
		list = append(list, string(name))
	}
	*m = list
	return nil
}

type UserAddress struct {
	Name string
	IP   string
}

type UserIP struct {
	List []UserAddress
}

func (*UserIP) Type() string {
	return "UserIP"
}

func (m *UserIP) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	for _, a := range m.List {
		if err := Name(a.Name).MarshalNMDC(enc, buf); err != nil {
			return err
		}
		buf.WriteByte(' ')
		buf.WriteString(a.IP)
		buf.WriteString("$$")
	}
	return nil
}

func (m *UserIP) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	data = bytes.TrimSuffix(data, []byte("\r"))
	data = bytes.TrimSuffix(data, []byte("$$"))
	sub := bytes.Split(data, []byte("$$"))
	m.List = make([]UserAddress, 0, len(sub))
	for _, s := range sub {
		var a UserAddress
		i := bytes.LastIndex(s, []byte(" "))
		if i >= 0 {
			a.IP = string(s[i+1:])
			s = s[:i]
		}
		var name Name
		if err := name.UnmarshalNMDC(dec, s); err != nil {
			return err
		}
		a.Name = string(name)
		m.List = append(m.List, a)
	}
	return nil
}
