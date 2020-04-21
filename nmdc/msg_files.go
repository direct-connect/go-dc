package nmdc

import (
	"bytes"
	"errors"
	"strconv"
)

func init() {
	RegisterMessage(&ADCGet{})
	RegisterMessage(&ADCSnd{})
	RegisterMessage(&Direction{})
	RegisterMessage(&MaxedOut{})
}

type ADCGet struct {
	ContentType     String
	Identifier      String
	Start           uint64
	Length          int64
	Compressed      bool
	DownloadedBytes *uint64
}

func (*ADCGet) Type() string {
	return "ADCGET"
}

func (m *ADCGet) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if err := m.ContentType.MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteByte(' ')
	if err := m.Identifier.MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteByte(' ')
	buf.WriteString(strconv.FormatUint(m.Start, 10))
	buf.WriteByte(' ')
	buf.WriteString(strconv.FormatInt(m.Length, 10))

	if m.Compressed {
		buf.Write([]byte(" ZL1"))
	}

	if m.DownloadedBytes != nil {
		buf.Write([]byte(" DB"))
		buf.WriteString(strconv.FormatUint(*m.DownloadedBytes, 10))
	}

	return nil
}

func (m *ADCGet) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCGet: missing separator after field 'ContentType'")
	}
	contentType, data := data[:i], data[i+1:]
	if err := m.ContentType.UnmarshalNMDC(dec, contentType); err != nil {
		return err
	}

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCGet: missing separator after field 'Identifier'")
	}
	identifier, data := data[:i], data[i+1:]
	if err := m.Identifier.UnmarshalNMDC(dec, identifier); err != nil {
		return errors.New("ADCGet: unable to parse field 2")
	}

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCGet: missing separator after field 'Start'")
	}
	start, data := data[:i], data[i+1:]
	var err error
	m.Start, err = parseUin64Trim(start)
	if err != nil {
		return errors.New("ADCGet: unable to parse field 'Start'")
	}

	i = bytes.IndexByte(data, ' ')
	var length []byte
	if i < 0 {
		length, data = data[:], nil
	} else {
		length, data = data[:i], data[i+1:]
	}
	if bytes.Equal(length, []byte("-1")) {
		m.Length = -1
	} else {
		lengthNum, err := parseUin64Trim(length)
		if err != nil {
			return errors.New("ADCGet: unable to parse field 'Lenght'")
		}
		m.Length = int64(lengthNum)
	}

	// optional fields / flags
	for len(data) > 0 {
		i = bytes.IndexByte(data, ' ')
		var field []byte
		if i < 0 {
			field, data = data[:], nil
		} else {
			field, data = data[:i], data[i+1:]
		}

		if bytes.Equal(field, []byte("ZL1")) {
			m.Compressed = true

		} else if bytes.HasPrefix(field, []byte("DB")) {
			field = field[2:]
			dlBytes, err := parseUin64Trim(field)
			if err != nil {
				return errors.New("ADCGet: unable to parse field 'DownloadedBytes'")
			}
			m.DownloadedBytes = &dlBytes
		}
	}

	return nil
}

type ADCSnd struct {
	ContentType String
	Identifier  String
	Start       uint64
	Length      uint64
	Compressed  bool
}

func (*ADCSnd) Type() string {
	return "ADCSND"
}

func (m *ADCSnd) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if err := m.ContentType.MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteByte(' ')
	if err := m.Identifier.MarshalNMDC(enc, buf); err != nil {
		return err
	}
	buf.WriteByte(' ')
	buf.WriteString(strconv.FormatUint(m.Start, 10))
	buf.WriteByte(' ')
	buf.WriteString(strconv.FormatUint(m.Length, 10))
	if m.Compressed {
		buf.Write([]byte(" ZL1"))
	}
	return nil
}

func (m *ADCSnd) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCSnd: missing separator after field 1")
	}
	contentType, data := data[:i], data[i+1:]
	if err := m.ContentType.UnmarshalNMDC(dec, contentType); err != nil {
		return err
	}

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCSnd: missing separator after field 2")
	}
	identifier, data := data[:i], data[i+1:]
	if err := m.Identifier.UnmarshalNMDC(dec, identifier); err != nil {
		return errors.New("ADCSnd: unable to parse field 2")
	}

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCSnd: missing separator after field 3")
	}
	start, data := data[:i], data[i+1:]
	var err error
	m.Start, err = parseUin64Trim(start)
	if err != nil {
		return errors.New("ADCSnd: unable to parse field 3")
	}

	i = bytes.IndexByte(data, ' ')
	var length []byte
	if i < 0 {
		length, data = data[:], nil
	} else {
		length, data = data[:i], data[i+1:]
	}
	m.Length, err = parseUin64Trim(length)
	if err != nil {
		return errors.New("ADCSnd: unable to parse field 4")
	}

	if len(data) > 0 {
		if !bytes.Equal(data, []byte("ZL1")) {
			return errors.New("ADCSnd: invalid field 5")
		}
		m.Compressed = true
	}

	return nil
}

type Direction struct {
	Upload bool
	Number uint
}

func (*Direction) Type() string {
	return "Direction"
}

func (m *Direction) MarshalNMDC(_ *TextEncoder, buf *bytes.Buffer) error {
	if m.Upload {
		buf.Write([]byte("Upload"))
	} else {
		buf.Write([]byte("Download"))
	}
	buf.WriteByte(' ')
	buf.WriteString(strconv.Itoa(int(m.Number)))
	return nil
}

func (m *Direction) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("Direction: missing field 1")
	}
	direction, data := data[:i], data[i+1:]
	if bytes.Equal(direction, []byte("Download")) {
		m.Upload = false
	} else if bytes.Equal(direction, []byte("Upload")) {
		m.Upload = true
	} else {
		return errors.New("Direction: invalid direction field")
	}

	number, err := parseUin64Trim(data)
	if err != nil {
		return errors.New("Direction: unable to parse field 2")
	}
	if number < 1 || number > 32767 {
		return errors.New("Direction: number outside range")
	}
	m.Number = uint(number)

	return nil
}

type MaxedOut struct {
	NoArgs
}

func (*MaxedOut) Type() string {
	return "MaxedOut"
}
