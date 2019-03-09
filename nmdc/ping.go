package nmdc

import (
	"bytes"
	"errors"
	"strconv"
	"strings"

	"github.com/direct-connect/go-dc"
)

func init() {
	RegisterMessage(&BotINFO{})
	RegisterMessage(&HubINFO{})
}

// BotINFO is sent by the pinger to the hub to get a HubINFO.
type BotINFO struct {
	String
}

func (*BotINFO) Type() string {
	return "BotINFO"
}

// HubINFO is a detailed hub information exposed only after receiving BotINFO.
type HubINFO struct {
	Name     string
	Host     string
	Desc     string
	I1       int // TODO
	I2       int // TODO
	I3       int // TODO
	I4       int // TODO
	Soft     dc.Software
	Owner    string
	State    string // TODO
	Encoding string
}

func (*HubINFO) Type() string {
	return "HubINFO"
}

func (m *HubINFO) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	const sep = '$'
	if err := String(m.Name).MarshalNMDC(enc, buf); err != nil {
		return err
	}

	buf.WriteByte(sep)
	buf.WriteString(m.Host)

	buf.WriteByte(sep)
	if err := String(m.Desc).MarshalNMDC(enc, buf); err != nil {
		return err
	}

	buf.WriteByte(sep)
	buf.WriteString(strconv.Itoa(m.I1))
	buf.WriteByte(sep)
	buf.WriteString(strconv.Itoa(m.I2))
	buf.WriteByte(sep)
	buf.WriteString(strconv.Itoa(m.I3))
	buf.WriteByte(sep)
	buf.WriteString(strconv.Itoa(m.I4))

	buf.WriteByte(sep)
	if m.Soft.Version == "" {
		buf.WriteString(m.Soft.Name)
	} else {
		buf.WriteString(m.Soft.Name)
		buf.WriteString(" ")
		buf.WriteString(m.Soft.Version)
	}

	buf.WriteByte(sep)
	if err := String(m.Owner).MarshalNMDC(enc, buf); err != nil {
		return err
	}

	buf.WriteByte(sep)
	if err := String(m.State).MarshalNMDC(enc, buf); err != nil {
		return err
	}

	buf.WriteByte(sep)
	buf.WriteString(m.Encoding)
	return nil
}

func (m *HubINFO) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	fields := bytes.SplitN(data, []byte("$"), 13)
	for i, field := range fields {
		switch i {
		case 0:
			var s String
			if err := s.UnmarshalNMDC(dec, field); err != nil {
				return err
			}
			m.Name = string(s)
		case 1:
			m.Host = string(field)
		case 2:
			var s String
			if err := s.UnmarshalNMDC(dec, field); err != nil {
				return err
			}
			m.Desc = string(s)
		case 3:
			i1, err := atoiTrim(field)
			if err != nil {
				return errors.New("invalid i1")
			}
			m.I1 = i1
		case 4:
			i2, err := atoiTrim(field)
			if err != nil {
				return errors.New("invalid i2")
			}
			m.I2 = i2
		case 5:
			i3, err := atoiTrim(field)
			if err != nil {
				return errors.New("invalid i3")
			}
			m.I3 = i3
		case 6:
			i4, err := atoiTrim(field)
			if err != nil {
				return errors.New("invalid i4")
			}
			m.I4 = i4
		case 7:
			soft := string(field)
			m.Soft.Name = soft
			if i := strings.LastIndexByte(soft, ' '); i >= 0 {
				m.Soft.Name = soft[:i]
				m.Soft.Version = soft[i+1:]
			}
		case 8:
			var s String
			if err := s.UnmarshalNMDC(dec, field); err != nil {
				return err
			}
			m.Owner = string(s)
		case 9:
			var s String
			if err := s.UnmarshalNMDC(dec, field); err != nil {
				return err
			}
			m.State = string(s)
		case 10:
			if len(fields) < 12 {
				m.Encoding = string(field)
			}
		default:
			// ignore everything else
		}
	}
	return nil
}
