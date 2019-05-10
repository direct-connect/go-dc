package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/direct-connect/go-dc/tiger"
)

func init() {
	RegisterMessage(&Search{})
	RegisterMessage(&SR{})
	RegisterMessage(&TTHSearchActive{})
	RegisterMessage(&TTHSearchPassive{})
}

type DataType uint

const (
	DataTypeAny        = DataType(1)
	DataTypeAudio      = DataType(2)
	DataTypeCompressed = DataType(3)
	DataTypeDocument   = DataType(4)
	DataTypeExecutable = DataType(5)
	DataTypePicture    = DataType(6)
	DataTypeVideo      = DataType(7)
	DataTypeFolders    = DataType(8)
	DataTypeTTH        = DataType(9)
	DataTypeDiskImage  = DataType(10)
	DataTypeComics     = DataType(11)
	DataTypeBook       = DataType(12)
	DataTypeMagnet     = DataType(13)
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

	var b [4 + tiger.Base32Size]byte
	b[0] = ' '
	if m.SizeRestricted {
		copy(b[1:], "T?")
		if m.IsMaxSize {
			b[3] = 'T'
		} else {
			b[3] = 'F'
		}
	} else {
		copy(b[1:], "F?T")
	}
	b[4] = '?'
	bi := strconv.AppendUint(b[:5], m.Size, 10)
	bi = append(bi, '?')
	if m.DataType == 0 {
		bi = strconv.AppendUint(bi, uint64(DataTypeAny), 10)
	} else {
		bi = strconv.AppendUint(bi, uint64(m.DataType), 10)
	}
	bi = append(bi, '?')
	buf.Write(bi)

	if m.DataType == DataTypeTTH {
		if m.TTH == nil {
			return errors.New("invalid TTH pointer")
		}
		copy(b[:4], "TTH:")
		if err := m.TTH.MarshalBase32(b[4:]); err != nil {
			return err
		}
		buf.Write(b[:])
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
	fields := bytes.SplitN(data, []byte{'?'}, 5)
	if len(fields) < 5 {
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
		if field[0] == '-' {
			// some clients send SizeRestricted=true and Size=-1
			_, err = parseUin64Trim(field[1:])
			if err != nil {
				return err
			}
			m.SizeRestricted = false
			m.Size = 0
		} else {
			m.Size, err = parseUin64Trim(field)
			if err != nil {
				return err
			}
		}
	}

	next()
	typ, err := parseUin64Trim(field)
	if err != nil {
		return err
	}
	m.DataType = DataType(typ)

	next()
	if m.DataType == DataTypeTTH {
		const tthPref = "TTH:"
		if !bytes.HasPrefix(field, []byte(tthPref)) {
			return fmt.Errorf("invalid TTH search")
		}
		hash := field[len(tthPref):]
		if n := len(hash); n != 0 && hash[n-1] == '$' {
			hash = hash[:n-1]
		}
		m.TTH = new(TTH)
		err := m.TTH.UnmarshalBase32(hash)
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
	buf.WriteByte(' ')
	for i, p := range m.Path {
		if i != 0 {
			buf.WriteByte('\\')
		}
		if err := String(p).MarshalNMDC(enc, buf); err != nil {
			return err
		}
	}
	var b [4 + tiger.Base32Size]byte
	if !m.IsDir {
		b[0] = srSep
		bi := strconv.AppendUint(b[:1], m.Size, 10)
		buf.Write(bi)
	}
	bi := append(b[:0], ' ')
	bi = strconv.AppendInt(bi, int64(m.FreeSlots), 10)
	bi = append(bi, '/')
	bi = strconv.AppendInt(bi, int64(m.TotalSlots), 10)
	bi = append(bi, srSep)
	buf.Write(bi)

	if m.TTH != nil {
		copy(b[:4], "TTH:")
		if err := m.TTH.MarshalBase32(b[4:]); err != nil {
			return err
		}
		buf.Write(b[:])
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

	// The path may contain spaces, but we have 0x05 separators, right?
	// However, for directories, there is no 0x05 separator between the path and slots.

	i = bytes.IndexByte(data, srSep)
	if i < 0 {
		return errors.New("invalid SR command: missing separator")
	}

	// this should contain either path or path and slots
	maybePath := data[:i]
	data = data[i+1:]

	// try to locate slot separator '/', scanning backward
	hasSep := false
	i = -1
	for j := len(maybePath) - 1; j >= 0; j-- {
		r := maybePath[j]
		if r == '/' {
			hasSep = true
		} else if r < '0' || r > '9' {
			i = j
			break
		}
	}
	var (
		path  []byte
		slots []byte
	)
	if hasSep && i >= 0 {
		// directory result - slots are in the path
		m.IsDir = true
		m.Size = 0
		path = maybePath[:i]
		slots = maybePath[i+1:]
	} else {
		// file result - slots and size are after the next 0x05
		m.IsDir = false
		path = maybePath
		i = bytes.IndexByte(data, srSep)
		if i < 0 {
			return errors.New("invalid SR command: missing size")
		}
		sizeAndSlots := data[:i]
		data = data[i+1:]
		i = bytes.IndexByte(sizeAndSlots, ' ')
		if i < 0 {
			return errors.New("invalid SR command: missing size separator")
		}
		m.Size, err = parseUin64Trim(sizeAndSlots[:i])
		if err != nil {
			return err
		}
		slots = sizeAndSlots[i+1:]
	}
	var spath String
	if err := spath.UnmarshalNMDC(dec, path); err != nil {
		return err
	}
	m.Path = strings.Split(string(path), "\\")

	i = bytes.IndexByte(slots, '/')
	if i < 0 {
		return errors.New("invalid SR command: missing slots separator")
	}
	m.FreeSlots, err = atoiTrim(slots[:i])
	if err != nil {
		return err
	}
	m.TotalSlots, err = atoiTrim(slots[i+1:])
	if err != nil {
		return err
	}

	if bytes.HasPrefix(data, []byte("TTH:")) {
		data = data[4:]
		i = tiger.Base32Size
		if i+1 >= len(data) || data[i] != ' ' || data[i+1] != '(' {
			return errors.New("invalid SR command: invalid TTH search")
		}
		res := data[:i]
		data = data[i+2:]
		m.TTH = new(TTH)
		err = m.TTH.UnmarshalBase32(res)
		if err != nil {
			return err
		}
	} else {
		i = bytes.Index(data, []byte(" ("))
		if i < 0 {
			return errors.New("invalid SR command: missing TTH or hub name")
		}
		res := data[:i]
		data = data[i+2:]
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
	var b [tiger.Base32Size + 1]byte
	if err := m.TTH.MarshalBase32(b[:]); err != nil {
		return err
	}
	b[len(b)-1] = ' '
	buf.Write(b[:])
	buf.WriteString(m.Address)
	return nil
}

func (m *TTHSearchActive) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	const i = tiger.Base32Size
	if i >= len(data) || data[i] != ' ' {
		return errors.New("missing separator in SA command")
	}
	if err := m.TTH.UnmarshalBase32(data[:i]); err != nil {
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
	var b [tiger.Base32Size + 1]byte
	if err := m.TTH.MarshalBase32(b[:]); err != nil {
		return err
	}
	b[len(b)-1] = ' '
	buf.Write(b[:])
	if err := Name(m.User).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	return nil
}

func (m *TTHSearchPassive) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	const i = tiger.Base32Size
	if i >= len(data) || data[i] != ' ' {
		return errors.New("missing separator in SP command")
	}
	if err := m.TTH.UnmarshalBase32(data[:i]); err != nil {
		return err
	}
	var name Name
	if err := name.UnmarshalNMDC(dec, data[i+1:]); err != nil {
		return err
	}
	m.User = string(name)
	return nil
}
