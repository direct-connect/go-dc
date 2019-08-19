package adc

import (
	"bytes"
	"errors"
	"strconv"
)

const (
	FileListBZIP = "files.xml.bz2"
)

func init() {
	RegisterMessage(GetInfoRequest{})
	RegisterMessage(GetRequest{})
	RegisterMessage(GetResponse{})
}

type GetInfoRequest struct {
	Type string `adc:"#"`
	Path string `adc:"#"`
}

func (GetInfoRequest) Cmd() MsgType {
	return MsgType{'G', 'F', 'I'}
}

var (
	_ Marshaler   = GetRequest{}
	_ Unmarshaler = (*GetRequest)(nil)
)

type GetRequest struct {
	Type       string
	Path       string
	Start      int64
	Bytes      int64
	Compressed bool
}

func (GetRequest) Cmd() MsgType {
	return MsgType{'G', 'E', 'T'}
}

func (m GetRequest) MarshalADC(buf *bytes.Buffer) error {
	buf.Write(escape(m.Type))
	buf.WriteByte(' ')
	buf.Write(escape(m.Path))
	buf.WriteByte(' ')
	buf.WriteString(strconv.FormatInt(m.Start, 10))
	buf.WriteByte(' ')
	buf.WriteString(strconv.FormatInt(m.Bytes, 10))
	if m.Compressed {
		buf.Write([]byte(" ZL1"))
	}
	return nil
}

func (m *GetRequest) UnmarshalADC(data []byte) error {
	i := bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCGET: missing separator after field 1")
	}
	typ, data := data[:i], data[i+1:]
	m.Type = unescape(typ)

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCGET: missing separator after field 2")
	}
	path, data := data[:i], data[i+1:]
	m.Path = unescape(path)

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("ADCGET: missing separator after field 3")
	}
	start, data := data[:i], data[i+1:]
	var err error
	m.Start, err = strconv.ParseInt(string(start), 10, 64)
	if err != nil {
		return errors.New("ADCGET: unable to parse field 3")
	}

	i = bytes.IndexByte(data, ' ')
	var length []byte
	if i < 0 {
		length, data = data[:], nil
	} else {
		length, data = data[:i], data[i+1:]
	}
	m.Bytes, err = strconv.ParseInt(string(length), 10, 64)
	if err != nil {
		return errors.New("ADCGET: unable to parse field 4")
	}

	if len(data) > 0 {
		if !bytes.Equal(data, []byte("ZL1")) {
			return errors.New("ADCGET: invalid field 5")
		}
		m.Compressed = true
	}

	return nil
}

type GetResponse GetRequest

func (GetResponse) Cmd() MsgType {
	return MsgType{'S', 'N', 'D'}
}

func (m GetResponse) MarshalADC(buf *bytes.Buffer) error {
	return GetRequest(m).MarshalADC(buf)
}

func (m *GetResponse) UnmarshalADC(data []byte) error {
	return (*GetRequest)(m).UnmarshalADC(data)
}
