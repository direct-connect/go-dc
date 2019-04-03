package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func init() {
	RegisterMessage(&Search{})
	RegisterMessage(&SR{})
	RegisterMessage(&TTHSearchActive{})
	RegisterMessage(&TTHSearchPassive{})
}

type DataType byte

const (
	DataTypeAny        = DataType('1')
	DataTypeAudio      = DataType('2')
	DataTypeCompressed = DataType('3')
	DataTypeDocument   = DataType('4')
	DataTypeExecutable = DataType('5')
	DataTypePicture    = DataType('6')
	DataTypeVideo      = DataType('7')
	DataTypeFolders    = DataType('8')
	DataTypeTTH        = DataType('9')
)

type Search struct {
	Address string
	User    string

	SizeRestricted bool
	IsMaxSize      bool
	Size           uint64
	DataType       DataType

	Pattern string
	TTH     *TTH
}

func (*Search) Type() string {
	return "Search"
}

func (m *Search) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if m.Address != "" {
		buf.WriteString(m.Address)
	} else {
		buf.WriteString("Hub:")
		if err := Name(m.User).MarshalNMDC(enc, buf); err != nil {
			return err
		}
	}
	buf.WriteByte(' ')

	if m.SizeRestricted {
		buf.WriteString("T?")
		if m.IsMaxSize {
			buf.WriteByte('T')
		} else {
			buf.WriteByte('F')
		}
	} else {
		buf.WriteString("F?T")
	}

	buf.WriteByte('?')
	buf.WriteString(strconv.FormatUint(m.Size, 10))

	buf.WriteByte('?')
	if m.DataType == 0 {
		buf.WriteByte(byte(DataTypeAny))
	} else {
		buf.WriteByte(byte(m.DataType))
	}

	buf.WriteByte('?')
	if m.DataType == DataTypeTTH {
		if m.TTH == nil {
			return fmt.Errorf("invalid TTH pointer")
		}
		buf.WriteString("TTH:")
		buf.WriteString(m.TTH.Base32())
	} else {
		buf2 := bytes.NewBuffer(nil)
		if err := String(m.Pattern).MarshalNMDC(enc, buf2); err != nil {
			return err
		}
		pattern := bytes.Replace(buf2.Bytes(), []byte(" "), []byte("$"), -1)
		buf.Write(pattern)
	}
	return nil
}

func (m *Search) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("invalid search command")
	}
	field0 := data[:i]
	field1 := data[i+1:]
	const namePref = "Hub:"
	if val := field0; bytes.HasPrefix(val, []byte(namePref)) {
		var name Name
		err := name.UnmarshalNMDC(dec, val[len(namePref):])
		if err != nil {
			return err
		}
		m.User = string(name)
	} else {
		m.Address = string(field0)
	}
	return m.unmarshalString(dec, field1)
}

func (m *Search) unmarshalString(dec *TextDecoder, data []byte) error {
	fields, ok := splitN(data, '?', 5)
	if !ok {
		return errors.New("invalid search string")
	}
	var field []byte
	next := func() {
		field = fields[0]
		fields = fields[1:]
	}

	var err error

	next()
	m.SizeRestricted, err = unmarshalBoolFlag(field)
	if err != nil {
		return err
	}

	next()
	m.IsMaxSize, err = unmarshalBoolFlag(field)
	if err != nil {
		return err
	}

	next()
	if len(field) != 0 {
		m.Size, err = parseUin64Trim(field)
		if err != nil {
			return err
		}
	}

	next()
	if len(field) != 1 {
		return fmt.Errorf("invalid data type: %q", string(field))
	}
	m.DataType = DataType(field[0])

	next()
	if m.DataType == DataTypeTTH {
		const tthPref = "TTH:"
		if !bytes.HasPrefix(field, []byte(tthPref)) {
			return fmt.Errorf("invalid TTH search")
		}
		hash := field[len(tthPref):]
		m.TTH = new(TTH)
		err := m.TTH.FromBase32(string(hash))
		if err != nil {
			return err
		}
	} else {
		var str String
		err := str.UnmarshalNMDC(dec, field)
		if err != nil {
			return err
		}
		m.Pattern = strings.Replace(string(str), "$", " ", -1)
	}
	return nil
}

func unmarshalBoolFlag(data []byte) (bool, error) {
	if len([]byte(data)) != 1 {
		return false, fmt.Errorf("invalid bool flag")
	}
	if data[0] == 'T' {
		return true, nil
	}
	if data[0] == 'F' {
		return false, nil
	}
	return false, fmt.Errorf("invalid bool flag")
}

type SR struct {
	From       string
	Path       []string
	IsDir      bool
	Size       uint64 // only set for files
	FreeSlots  int
	TotalSlots int
	HubName    string
	TTH        *TTH
	HubAddress string
	To         string
}

const srSep = 0x05

func (*SR) Type() string {
	return "SR"
}

func (m *SR) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if err := Name(m.From).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	if len(m.Path) == 0 {
		return errors.New("invalid SR command: empty path")
	}
	path := strings.Join(m.Path, "\\")
	buf.WriteByte(' ')
	if err := String(path).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	if !m.IsDir {
		buf.WriteByte(srSep)
		buf.WriteString(strconv.FormatUint(m.Size, 10))
	}
	buf.WriteByte(' ')
	buf.WriteString(strconv.Itoa(m.FreeSlots))
	buf.WriteByte('/')
	buf.WriteString(strconv.Itoa(m.TotalSlots))
	buf.WriteByte(srSep)
	if m.TTH != nil {
		buf.WriteString("TTH:")
		buf.WriteString(m.TTH.Base32())
	} else {
		// legacy
		if err := String(m.HubName).MarshalNMDC(enc, buf); err != nil {
			return err
		}
	}
	buf.WriteString(" (")
	buf.WriteString(m.HubAddress)
	buf.WriteByte(')')
	if m.To == "" {
		return nil
	}
	buf.WriteByte(srSep)
	if err := Name(m.To).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	return nil
}

func (m *SR) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("invalid SR command: missing name")
	}
	var name Name
	err := name.UnmarshalNMDC(dec, data[:i])
	if err != nil {
		return err
	}
	data = data[i+1:]
	m.From = string(name)

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("invalid SR command: missing size")
	}
	res := data[:i]
	data = data[i+1:]

	var path String
	if i = bytes.IndexByte(res, srSep); i >= 0 {
		if err := path.UnmarshalNMDC(dec, res[:i]); err != nil {
			return err
		}
		m.Path = strings.Split(string(path), "\\")
		m.Size, err = parseUin64Trim(res[i+1:])
		if err != nil {
			return err
		}
	} else {
		if err := path.UnmarshalNMDC(dec, res); err != nil {
			return err
		}
		m.Path = strings.Split(string(path), "\\")
		m.IsDir = true
	}

	i = bytes.IndexByte(data, srSep)
	if i < 0 {
		return errors.New("invalid SR command: missing slots")
	}
	res = data[:i]
	data = data[i+1:]

	i = bytes.IndexByte(res, '/')
	if i < 0 {
		return errors.New("invalid SR command: missing slots separator")
	}
	m.FreeSlots, err = atoiTrim(res[:i])
	if err != nil {
		return err
	}
	m.TotalSlots, err = atoiTrim(res[i+1:])
	if err != nil {
		return err
	}

	i = bytes.Index(data, []byte(" ("))
	if i < 0 {
		return errors.New("invalid SR command: missing TTH or hub name")
	}
	res = data[:i]
	data = data[i+2:]
	if bytes.HasPrefix(res, []byte("TTH:")) {
		m.TTH = new(TTH)
		err = m.TTH.FromBase32(string(res[4:]))
		if err != nil {
			return err
		}
	} else {
		var s String
		err = s.UnmarshalNMDC(dec, res)
		if err != nil {
			return err
		}
		m.HubName = string(s)
	}
	i = bytes.IndexByte(data, ')')
	if i < 0 {
		return errors.New("invalid SR command: missing hub address")
	}
	m.HubAddress = string(data[:i])
	data = data[i+1:]
	if len(data) == 0 {
		return nil
	}
	if data[0] != srSep || len(data) == 1 {
		return errors.New("invalid SR command: missing target")
	}
	data = data[1:]
	if err := name.UnmarshalNMDC(dec, data); err != nil {
		return err
	}
	m.To = string(name)
	return nil
}

// TTHSearchActive is added by TTHS extension.
type TTHSearchActive struct {
	TTH     TTH
	Address string
}

func (*TTHSearchActive) Type() string {
	return "SA"
}

func (m *TTHSearchActive) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	buf.WriteString(m.TTH.Base32())
	buf.WriteByte(' ')
	buf.WriteString(m.Address)
	return nil
}

func (m *TTHSearchActive) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("missing separator in SA command")
	}
	if err := m.TTH.FromBase32(string(data[:i])); err != nil {
		return err
	}
	m.Address = string(data[i+1:])
	return nil
}

// TTHSearchPassive is added by TTHS extension.
type TTHSearchPassive struct {
	TTH  TTH
	User string
}

func (*TTHSearchPassive) Type() string {
	return "SP"
}

func (m *TTHSearchPassive) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	buf.WriteString(m.TTH.Base32())
	buf.WriteByte(' ')
	buf.WriteString(m.User)
	return nil
}

func (m *TTHSearchPassive) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("missing separator in SP command")
	}
	if err := m.TTH.FromBase32(string(data[:i])); err != nil {
		return err
	}
	var name Name
	if err := name.UnmarshalNMDC(dec, data[i+1:]); err != nil {
		return err
	}
	m.User = string(name)
	return nil
}
