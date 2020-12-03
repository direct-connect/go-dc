package adc

import (
	"bytes"
	"errors"
	"fmt"
)

const lineDelim = '\n'

const (
	kindBroadcast = 'B'
	kindClient    = 'C'
	kindDirect    = 'D'
	kindEcho      = 'E'
	kindFeature   = 'F'
	kindHub       = 'H'
	kindInfo      = 'I'
	kindUDP       = 'U'
)

type MsgType [3]byte

func (s MsgType) String() string { return string(s[:]) }

type Packet interface {
	Kind() byte
	Message() Message
	SetMessage(m Message)
	DecodeMessage() error
	DecodeMessageTo(m Message) error
	MarshalPacketADC(buf *bytes.Buffer) error
	UnmarshalPacketADC(name MsgType, data []byte) error
}

type PeerPacket interface {
	Packet
	Source() SID
}

type TargetPacket interface {
	Packet
	Target() SID
}

func decodeMessage(ptr *Message) error {
	if ptr == nil {
		return nil
	}
	m := *ptr
	raw, ok := m.(*RawMessage)
	if !ok {
		return nil
	}
	var err error
	m, err = raw.Decode()
	if err != nil {
		return err
	}
	*ptr = m
	return nil
}

func decodeMessageTo(from, to Message) error {
	raw, ok := from.(*RawMessage)
	if !ok {
		return errors.New("expected raw message")
	} else if to.Cmd() != raw.Type {
		return errors.New("expected " + to.Cmd().String() + ", got " + raw.Type.String())
	}
	return Unmarshal(raw.Data, to)
}

// DecodePacket decodes ADC line. The message payload is a copy.
func DecodePacket(p []byte) (Packet, error) {
	m, err := DecodePacketRaw(p)
	if err != nil {
		return nil, err
	}
	if err = m.DecodeMessage(); err != nil {
		return nil, err
	}
	return m, nil
}

// DecodePacketRaw decodes ADC line and returns packet with a RawMessage.
// RawMessage data slice will point to p, thus it's the caller's responsibility to copy it.
func DecodePacketRaw(p []byte) (Packet, error) {
	if len(p) < 5 {
		return nil, fmt.Errorf("too short for command: '%s'", string(p))
	}
	if bytes.IndexByte(p, 0x00) >= 0 {
		return nil, errors.New("messages should not contain null characters")
	}
	kind := p[0]
	var m Packet
	switch kind {
	case kindInfo:
		m = &InfoPacket{}
	case kindHub:
		m = &HubPacket{}
	case kindEcho:
		m = &EchoPacket{}
	case kindDirect:
		m = &DirectPacket{}
	case kindBroadcast:
		m = &BroadcastPacket{}
	case kindFeature:
		m = &FeaturePacket{}
	case kindClient:
		m = &ClientPacket{}
	case kindUDP:
		m = &UDPPacket{}
	default:
		return nil, fmt.Errorf("unknown command kind: %c", p[0])
	}
	var cname MsgType
	copy(cname[:], p[1:4])
	p = p[4:]
	if len(p) == 0 || p[len(p)-1] != lineDelim {
		return nil, errors.New("expected line delimiter")
	}
	if p[0] == ' ' {
		p = p[1:]
	} else if p[0] != lineDelim {
		return nil, errors.New("expected name delimiter")
	}
	if err := m.UnmarshalPacketADC(cname, p); err != nil {
		return nil, err
	}
	return m, nil
}

type InfoPacket struct {
	Msg Message
}

func (*InfoPacket) Kind() byte {
	return kindInfo
}

func (p *InfoPacket) Message() Message {
	return p.Msg
}

func (p *InfoPacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *InfoPacket) DecodeMessage() error {
	// special case: hub info
	infoType := MsgType{'I', 'N', 'F'}
	if p.Msg.Cmd() == infoType {
		msg := HubInfo{}
		if err := Unmarshal(p.Msg.(*RawMessage).Data, &msg); err != nil {
			return err
		}
		p.Msg = msg
		return nil
	}

	return decodeMessage(&p.Msg)
}

func (p *InfoPacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *InfoPacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// IINF <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, len(cmd)+2)
	b[0] = p.Kind()
	i := 1
	i += copy(b[i:], cmd[:])
	b[i] = ' '
	buf.Write(b)

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *InfoPacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	if len(data) != 0 && data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	}
	data = data[:len(data)-1]
	if len(data) == 0 {
		data = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}

type HubPacket InfoPacket

func (*HubPacket) Kind() byte {
	return kindHub
}

func (p *HubPacket) Message() Message {
	return p.Msg
}

func (p *HubPacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *HubPacket) DecodeMessage() error {
	return decodeMessage(&p.Msg)
}

func (p *HubPacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *HubPacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// HINF <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, len(cmd)+2)
	b[0] = p.Kind()
	i := 1
	i += copy(b[i:], cmd[:])
	b[i] = ' '
	buf.Write(b)

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *HubPacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	if len(data) < 1 {
		return errors.New("short hub command")
	} else if data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	}
	data = data[:len(data)-1]
	if len(data) == 0 {
		data = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}

var (
	_ Packet     = (*BroadcastPacket)(nil)
	_ PeerPacket = (*BroadcastPacket)(nil)
)

type BroadcastPacket struct {
	ID  SID
	Msg Message
}

func (*BroadcastPacket) Kind() byte {
	return kindBroadcast
}

func (p *BroadcastPacket) Source() SID {
	return p.ID
}

func (p *BroadcastPacket) Message() Message {
	return p.Msg
}

func (p *BroadcastPacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *BroadcastPacket) DecodeMessage() error {
	return decodeMessage(&p.Msg)
}

func (p *BroadcastPacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *BroadcastPacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// BINF AAAA <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, 1+len(cmd)+6)
	b[0] = p.Kind()
	i := 1
	i += copy(b[i:], cmd[:])
	b[i] = ' '
	i++
	i += copy(b[i:], p.ID[:])
	b[i] = ' '
	buf.Write(b)

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *BroadcastPacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	if len(data) < 4 {
		return errors.New("short broadcast command")
	} else if data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	} else if len(data) > 4 && data[4] != ' ' && data[4] != lineDelim {
		return fmt.Errorf("separator expected: '%s'", string(data[:5]))
	}
	data = data[:len(data)-1]
	if err := p.ID.UnmarshalADC(data[0:4]); err != nil {
		return err
	}
	if len(data) > 5 {
		data = data[5:]
	} else {
		data = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}

var (
	_ Packet       = (*DirectPacket)(nil)
	_ PeerPacket   = (*DirectPacket)(nil)
	_ TargetPacket = (*DirectPacket)(nil)
)

type DirectPacket struct {
	ID  SID
	To  SID
	Msg Message
}

func (*DirectPacket) Kind() byte {
	return kindDirect
}

func (p *DirectPacket) Source() SID {
	return p.ID
}

func (p *DirectPacket) Target() SID {
	return p.To
}

func (p *DirectPacket) Message() Message {
	return p.Msg
}

func (p *DirectPacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *DirectPacket) DecodeMessage() error {
	return decodeMessage(&p.Msg)
}

func (p *DirectPacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *DirectPacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// DCTM AAAA BBBB <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, 1+len(cmd)+11)
	b[0] = p.Kind()
	i := 1

	i += copy(b[i:], cmd[:])
	b[i] = ' '
	i++

	i += copy(b[i:], p.ID[:])
	b[i] = ' '
	i++

	i += copy(b[i:], p.To[:])
	b[i] = ' '
	i++
	buf.Write(b)

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *DirectPacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	if len(data) < 9 {
		return fmt.Errorf("short direct command")
	} else if data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	} else if data[4] != ' ' {
		return fmt.Errorf("separator expected: '%s'", string(data[:9]))
	} else if len(data) > 9 && data[9] != ' ' && data[9] != lineDelim {
		return fmt.Errorf("separator expected: '%s'", string(data[:10]))
	}
	data = data[:len(data)-1]
	if err := p.ID.UnmarshalADC(data[0:4]); err != nil {
		return err
	}
	if err := p.To.UnmarshalADC(data[5:9]); err != nil {
		return err
	}
	if len(data) > 10 {
		data = data[10:]
	} else {
		data = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}

var (
	_ Packet       = (*EchoPacket)(nil)
	_ PeerPacket   = (*EchoPacket)(nil)
	_ TargetPacket = (*EchoPacket)(nil)
)

type EchoPacket DirectPacket

func (*EchoPacket) Kind() byte {
	return kindEcho
}

func (p *EchoPacket) Source() SID {
	return p.ID
}

func (p *EchoPacket) Target() SID {
	return p.To
}

func (p *EchoPacket) Message() Message {
	return p.Msg
}

func (p *EchoPacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *EchoPacket) DecodeMessage() error {
	return decodeMessage(&p.Msg)
}

func (p *EchoPacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *EchoPacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// EMSG AAAA BBBB <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, 1+len(cmd)+11)
	b[0] = p.Kind()
	i := 1

	i += copy(b[i:], cmd[:])
	b[i] = ' '
	i++

	i += copy(b[i:], p.ID[:])
	b[i] = ' '
	i++

	i += copy(b[i:], p.To[:])
	b[i] = ' '
	i++
	buf.Write(b)

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *EchoPacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	if len(data) < 9 {
		return fmt.Errorf("short echo command")
	} else if data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	} else if data[4] != ' ' {
		return fmt.Errorf("separator expected: '%s'", string(data[:9]))
	} else if len(data) > 9 && data[9] != ' ' && data[9] != lineDelim {
		return fmt.Errorf("separator expected: '%s'", string(data[:10]))
	}
	data = data[:len(data)-1]
	if err := p.ID.UnmarshalADC(data[0:4]); err != nil {
		return err
	}
	if err := p.To.UnmarshalADC(data[5:9]); err != nil {
		return err
	}
	if len(data) > 10 {
		data = data[10:]
	} else {
		data = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}

type ClientPacket struct {
	Msg Message
}

func (*ClientPacket) Kind() byte {
	return kindClient
}

func (p *ClientPacket) Message() Message {
	return p.Msg
}

func (p *ClientPacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *ClientPacket) DecodeMessage() error {
	return decodeMessage(&p.Msg)
}

func (p *ClientPacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *ClientPacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// CINF <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, len(cmd)+2)
	b[0] = p.Kind()
	i := 1
	i += copy(b[i:], cmd[:])
	b[i] = ' '
	buf.Write(b)

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *ClientPacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	if len(data) < 1 {
		return errors.New("short client command")
	} else if data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	}
	data = data[:len(data)-1]
	if len(data) == 0 {
		data = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}

var (
	_ Packet     = (*FeaturePacket)(nil)
	_ PeerPacket = (*FeaturePacket)(nil)
)

type FeatureSel struct {
	Fea Feature
	Sel bool
}

type FeaturePacket struct {
	ID  SID
	Sel []FeatureSel
	Msg Message
}

func (*FeaturePacket) Kind() byte {
	return kindFeature
}

func (p *FeaturePacket) Source() SID {
	return p.ID
}

func (p *FeaturePacket) Message() Message {
	return p.Msg
}

func (p *FeaturePacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *FeaturePacket) DecodeMessage() error {
	return decodeMessage(&p.Msg)
}

func (p *FeaturePacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *FeaturePacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// FSCH AAAA +SEGA -NAT0 <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, 1+len(cmd)+6)
	b[0] = p.Kind()
	i := 1
	i += copy(b[i:], cmd[:])
	b[i] = ' '
	i++
	i += copy(b[i:], p.ID[:])
	b[i] = ' '
	buf.Write(b)

	for _, f := range p.Sel {
		i = 0
		if f.Sel {
			b[i] = '+'
		} else {
			b[i] = '-'
		}
		i++
		i += copy(b[i:], f.Fea[:])
		b[i] = ' '
		i++
		buf.Write(b[:i])
	}

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *FeaturePacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("short feature command")
	} else if data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	} else if len(data) > 4 && data[4] != ' ' && data[4] != lineDelim {
		return fmt.Errorf("separator expected: '%s'", string(data[:5]))
	}
	data = data[:len(data)-1]
	if err := p.ID.UnmarshalADC(data[0:4]); err != nil {
		return err
	}
	if len(data) > 5 {
		data = data[5:]
	} else {
		data = nil
	}
	p.Sel = make([]FeatureSel, 0, len(data)/5)
	for i := 0; i < len(data); i++ {
		if data[i] == '+' {
			if f := data[i:]; len(f) < 5 {
				return fmt.Errorf("short feature: '%s'", string(data[i:i+5]))
			}
			var fea Feature
			copy(fea[:], data[i+1:i+5])
			p.Sel = append(p.Sel, FeatureSel{Fea: fea, Sel: true})
			i += 4
		} else if data[i] == '-' {
			if f := data[i:]; len(f) < 5 {
				return fmt.Errorf("short feature: '%s'", string(data[i:i+5]))
			}
			var fea Feature
			copy(fea[:], data[i+1:i+5])
			p.Sel = append(p.Sel, FeatureSel{Fea: fea, Sel: false})
			i += 4
		} else if data[i] == ' ' {
			data = data[i:]
			i = 0
		} else {
			data = data[i:]
			break
		}
		if i+1 == len(data) {
			data = nil
		}
	}
	if len(p.Sel) == 0 {
		p.Sel = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}

type UDPPacket struct {
	ID  CID
	Msg Message
}

func (*UDPPacket) Kind() byte {
	return kindUDP
}

func (p *UDPPacket) Message() Message {
	return p.Msg
}

func (p *UDPPacket) SetMessage(m Message) {
	p.Msg = m
}

func (p *UDPPacket) DecodeMessage() error {
	return decodeMessage(&p.Msg)
}

func (p *UDPPacket) DecodeMessageTo(m Message) error {
	return decodeMessageTo(p.Msg, m)
}

func (p *UDPPacket) MarshalPacketADC(buf *bytes.Buffer) error {
	// UINF <CID> <data>0x0a
	cmd := p.Msg.Cmd()

	b := make([]byte, 1+len(cmd)+39+2)
	b[0] = p.Kind()
	i := 1
	i += copy(b[i:], cmd[:])
	b[i] = ' '
	i++
	i += copy(b[i:], p.ID.Base32())
	b[i] = ' '
	buf.Write(b)

	i = buf.Len()
	if err := Marshal(buf, p.Msg); err != nil {
		return err
	}
	if i == buf.Len() {
		// empty payload - replace space with a separator
		buf.Bytes()[i-1] = lineDelim
	} else {
		buf.WriteByte(lineDelim)
	}
	return nil
}

func (p *UDPPacket) UnmarshalPacketADC(name MsgType, data []byte) error {
	const l = 39 // len of CID in base32
	if len(data) < l {
		return errors.New("short upd command")
	} else if data[len(data)-1] != lineDelim {
		return errors.New("invalid packet delimiter")
	} else if len(data) > l && data[l] != ' ' && data[l] != lineDelim {
		return fmt.Errorf("separator expected: '%s'", string(data[:l+1]))
	}
	data = data[:len(data)-1]
	if err := p.ID.FromBase32(string(data[0:l])); err != nil {
		return fmt.Errorf("wrong CID in upd command: %v", err)
	}
	if len(data) > l+1 {
		data = data[l+1:]
	} else {
		data = nil
	}
	p.Msg = &RawMessage{Type: name, Data: data}
	return nil
}
