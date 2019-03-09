package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/direct-connect/go-dc"
)

func init() {
	RegisterMessage(&ValidateNick{})
	RegisterMessage(&ValidateDenide{})
	RegisterMessage(&MyNick{})
	RegisterMessage(&Quit{})
	RegisterMessage(&MyINFO{})
	RegisterMessage(&UserIP{})
}

// ValidateNick is sent from the client to the hub as a request to enter with a specific
// user name.
//
// The hub will send Hello in case of success or ValidateDenide in case of an error.
type ValidateNick struct {
	Name
}

func (*ValidateNick) Type() string {
	return "ValidateNick"
}

type ValidateDenide struct {
	Name
}

func (*ValidateDenide) Type() string {
	return "ValidateDenide"
}

// MyNick is sent in C-C connections for clients to be able to identify each other.
type MyNick struct {
	Name
}

func (*MyNick) Type() string {
	return "MyNick"
}

// Quit is a notification about user quiting the hub.
type Quit struct {
	Name
}

func (*Quit) Type() string {
	return "Quit"
}

type UserMode byte

const (
	UserModeUnknown = UserMode(0)
	UserModeActive  = UserMode('A')
	UserModePassive = UserMode('P')
	UserModeSOCKS5  = UserMode('5')
)

type UserFlag byte

func (f UserFlag) IsSet(f2 UserFlag) bool {
	return f&f2 != 0
}

const (
	FlagStatusNormal   = UserFlag(0x01)
	FlagStatusAway     = UserFlag(0x02)
	FlagStatusServer   = UserFlag(0x04)
	FlagStatusFireball = UserFlag(0x08)
	FlagTLSDownload    = UserFlag(0x10)
	FlagTLSUpload      = UserFlag(0x20)
	FlagIPv4           = UserFlag(0x40)
	FlagIPv6           = UserFlag(0x80)

	FlagTLS = FlagTLSUpload | FlagTLSDownload
)

// Used by some clients to set a different icon.
const (
	ConnSpeedModem  = "1"
	ConnSpeedServer = "1000"
)

type MyINFO struct {
	Name           string
	Desc           string
	Client         dc.Software
	Mode           UserMode
	HubsNormal     int
	HubsRegistered int
	HubsOperator   int
	Slots          int
	Extra          map[string]string
	Conn           string
	Flag           UserFlag
	Email          string
	ShareSize      uint64
}

func (*MyINFO) Type() string {
	return "MyINFO"
}

func (m *MyINFO) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	buf.WriteString("$ALL ")
	if err := Name(m.Name).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteString(" ")
	if err := String(m.Desc).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteString("<")
	buf.WriteString(m.Client.Name)
	buf.WriteString(" ")

	buf.WriteString("V:")
	buf.WriteString(m.Client.Version)

	buf.WriteString(",M:")
	if m.Mode != UserModeUnknown && m.Mode != ' ' {
		buf.WriteByte(byte(m.Mode))
	} else {
		buf.WriteByte(' ')
	}

	buf.WriteString(",H:")
	buf.WriteString(strconv.Itoa(m.HubsNormal))
	buf.WriteString("/")
	buf.WriteString(strconv.Itoa(m.HubsRegistered))
	buf.WriteString("/")
	buf.WriteString(strconv.Itoa(m.HubsOperator))

	buf.WriteString(",S:")
	buf.WriteString(strconv.Itoa(m.Slots))

	for name, value := range m.Extra {
		buf.WriteString(",")
		buf.WriteString(name)
		buf.WriteString(":")
		buf.WriteString(value)
	}

	buf.WriteString(">")
	buf.WriteString("$ $")
	buf.WriteString(m.Conn)
	if m.Flag == 0 {
		buf.WriteByte(byte(FlagStatusNormal))
	} else {
		buf.WriteByte(byte(m.Flag))
	}
	buf.WriteString("$")
	buf.WriteString(m.Email)
	buf.WriteString("$")
	buf.WriteString(strconv.FormatUint(m.ShareSize, 10))
	buf.WriteString("$")
	return nil
}

func (m *MyINFO) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	const infoPref = "$ALL "
	if !bytes.HasPrefix(data, []byte(infoPref)) {
		return errors.New("invalid info command: wrong prefix")
	}
	data = data[len(infoPref):]

	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("invalid info command: no separators")
	}
	var name Name
	if err := name.UnmarshalNMDC(dec, data[:i]); err != nil {
		return err
	}
	data = data[i+1:]
	m.Name = string(name)

	n := bytes.Count(data, []byte{'$'})
	if n != 5 {
		return fmt.Errorf("invalid info command: %q", string(data))
	}

	var field []byte
	next := func() {
		i := bytes.IndexByte(data, '$')
		if i < 0 {
			field = data
			data = nil
		} else {
			field = data[:i]
			data = data[i+1:]
		}
	}

	next()
	m.Mode = UserModeUnknown
	hasTag := false
	var desc []byte
	i = bytes.IndexByte(field, '<')
	if i < 0 {
		desc = field
	} else {
		hasTag = true
		desc = field[:i]
		tag := field[i+1:]
		if len(tag) == 0 {
			return errors.New("empty info tag")
		}
		if tag[len(tag)-1] == '>' {
			tag = tag[:len(tag)-1]
		}
		if err := m.unmarshalTag(tag); err != nil {
			return err
		}
	}
	var s String
	if err := s.UnmarshalNMDC(dec, desc); err != nil {
		return err
	}
	m.Desc = string(s)

	next()
	if len(field) != 1 {
		return fmt.Errorf("unknown leacy user mode: %q", string(field))
	}
	if !hasTag {
		m.Mode = UserMode(field[0])
		if m.Mode == ' ' {
			m.Mode = UserModeUnknown
		}
	}

	next()
	if len(field) > 0 {
		l := len(field)
		m.Flag = UserFlag(field[l-1])
		m.Conn = string(field[:l-1])
	}

	next()
	m.Email = string(field)

	next()
	if len(field) != 0 {
		// TODO: add strict mode that will verify this
		size, _ := parseUin64Trim(field)
		m.ShareSize = size
	}
	return nil
}

func (m *MyINFO) unmarshalTag(tag []byte) error {
	var client []byte
	i := bytes.Index(tag, []byte(" V:"))
	if i < 0 {
		i = bytes.Index(tag, []byte(" v:"))
	}
	if i >= 0 {
		client = tag[:i]
		tag = tag[i+1:]
	}
	var err error
	m.Extra = nil
	for r := 0; len(tag) > 0; r++ {
		i = bytes.IndexByte(tag, ',')
		field := tag
		if i >= 0 {
			field = tag[:i]
			tag = tag[i+1:]
		} else {
			tag = nil
		}
		if len(field) == 0 {
			continue
		}
		i = bytes.IndexByte(field, ':')
		if r == 0 && i < 0 {
			client = field
			continue
		}
		if i <= 0 {
			return fmt.Errorf("unknown field name in tag: %q", field)
		}
		bkey := field[:i]
		val := field[i+1:]
		if len(bkey) != 1 {
			if m.Extra == nil {
				m.Extra = make(map[string]string)
			}
			m.Extra[string(bkey)] = string(val)
			continue
		}
		ckey := bkey[0]
		if ckey > 'Z' {
			ckey -= 'a' - 'A'
		}
		switch ckey {
		case 'V':
			m.Client.Version = string(val)
		case 'M':
			if len(val) == 1 {
				m.Mode = UserMode(val[0])
			} else {
				m.Mode = UserModeUnknown
			}
		case 'H':
			if len(val) == 0 {
				m.HubsNormal = 1
				continue
			}
			hubs, ok := splitN(val, '/', 3)
			if !ok {
				return fmt.Errorf("invalid hubs counts: %q", string(val))
			}
			m.HubsNormal, err = atoiTrim(hubs[0])
			if err != nil {
				return fmt.Errorf("invalid info hubs normal: %v", err)
			}
			m.HubsRegistered, err = atoiTrim(hubs[1])
			if err != nil {
				return fmt.Errorf("invalid info hubs registered: %v", err)
			}
			m.HubsOperator, err = atoiTrim(hubs[2])
			if err != nil {
				return fmt.Errorf("invalid info hubs operator: %v", err)
			}
		case 'S':
			m.Slots, err = atoiTrim(val)
			if err != nil {
				return fmt.Errorf("invalid slots: %q", string(val))
			}
		default:
			if m.Extra == nil {
				m.Extra = make(map[string]string)
			}
			m.Extra[string(ckey)] = string(val)
		}
	}
	m.Client.Name = string(client)
	return nil
}

type UserIP struct {
	Name string
	IP   string
}

func (*UserIP) Type() string {
	return "UserIP"
}

func (m *UserIP) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if err := Name(m.Name).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteString(" ")
	buf.WriteString(m.IP)
	buf.WriteString("$$")
	return nil
}

func (m *UserIP) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	data = bytes.TrimSuffix(data, []byte("$$"))
	i := bytes.LastIndex(data, []byte(" "))
	if i >= 0 {
		m.IP = string(data[i+1:])
		data = data[:i]
	}
	var name Name
	if err := name.UnmarshalNMDC(dec, data); err != nil {
		return err
	}
	m.Name = string(name)
	return nil
}
