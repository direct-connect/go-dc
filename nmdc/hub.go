package nmdc

import (
	"bytes"
)

func init() {
	RegisterMessage(&Hello{})
	RegisterMessage(&ZOn{})
	RegisterMessage(&Version{})
	RegisterMessage(&HubName{})
	RegisterMessage(&HubTopic{})
	RegisterMessage(&FailOver{})
}

type Hello struct {
	Name
}

func (*Hello) Type() string {
	return "Hello"
}

type HubName struct {
	String
}

func (*HubName) Type() string {
	return "HubName"
}

type Version struct {
	Vers string
}

func (*Version) Type() string {
	return "Version"
}

func (m *Version) MarshalNMDC(_ *TextEncoder, buf *bytes.Buffer) error {
	buf.WriteString(m.Vers)
	return nil
}

func (m *Version) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	m.Vers = string(data)
	return nil
}

type HubTopic struct {
	Text string
}

func (*HubTopic) Type() string {
	return "HubTopic"
}

func (m *HubTopic) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	return String(m.Text).MarshalNMDC(enc, buf)
}

func (m *HubTopic) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	var s String
	if err := s.UnmarshalNMDC(dec, data); err != nil {
		return err
	}
	m.Text = string(s)
	return nil
}

type FailOver struct {
	Host []string
}

func (*FailOver) Type() string {
	return "FailOver"
}

func (m *FailOver) MarshalNMDC(_ *TextEncoder, buf *bytes.Buffer) error {
	for i, s := range m.Host {
		if i != 0 {
			buf.WriteString(",")
		}
		buf.WriteString(s)
	}
	return nil
}

func (m *FailOver) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	hosts := bytes.Split(data, []byte(","))
	m.Host = make([]string, 0, len(hosts))
	for _, host := range hosts {
		m.Host = append(m.Host, string(host))
	}
	return nil
}

type ZOn struct {
	NoArgs
}

func (*ZOn) Type() string {
	return "ZOn"
}
