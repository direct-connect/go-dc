package adc

import (
	"errors"
	"fmt"
	"io"

	"github.com/direct-connect/go-dc/lineproto"
)

func NewReader(r io.Reader) *Reader {
	return &Reader{
		Reader: lineproto.NewReader(r, lineDelim),
	}
}

// Reader is not safe for concurrent use.
type Reader struct {
	*lineproto.Reader

	// OnKeepAlive is called when an empty (keep-alive) message is received.
	OnKeepAlive func() error
}

// ReadPacket reads and decodes a single ADC command.
func (r *Reader) ReadPacket() (Packet, error) {
	p, err := r.readPacket()
	if err != nil {
		return nil, err
	}
	return DecodePacket(p)
}

// ReadPacketRaw reads and decodes a single ADC command. The caller must copy the payload.
func (r *Reader) ReadPacketRaw() (Packet, error) {
	p, err := r.readPacket()
	if err != nil {
		return nil, err
	}
	return DecodePacketRaw(p)
}

func (r *Reader) ReadInfo() (Message, error) {
	cmd, err := r.readPacketRaw()
	if err != nil {
		return nil, err
	}
	cc, ok := cmd.(*InfoPacket)
	if !ok {
		return nil, fmt.Errorf("expected info command, got: %#v", cmd)
	}
	if raw, ok := cc.Msg.(*RawMessage); ok {
		return raw.Decode()
	}
	return cc.Msg, nil
}

func (r *Reader) ReadClient() (Message, error) {
	cmd, err := r.readPacketRaw()
	if err != nil {
		return nil, err
	}
	cc, ok := cmd.(*ClientPacket)
	if !ok {
		return nil, fmt.Errorf("expected client command, got: %#v", cmd)
	}
	if raw, ok := cc.Msg.(*RawMessage); ok {
		return raw.Decode()
	}
	return cc.Msg, nil
}

func (r *Reader) readPacketRaw() (Packet, error) {
	p, err := r.readPacket()
	if err != nil {
		return nil, err
	}
	return DecodePacketRaw(p)
}

// readPacket reads a single ADC packet (separated by 0x0a byte) without decoding it.
func (r *Reader) readPacket() ([]byte, error) {
	for {
		s, err := r.Reader.ReadLine()
		if err != nil {
			return nil, err
		} else if len(s) == 0 || s[len(s)-1] != lineDelim {
			return nil, errors.New("invalid packet delimiter")
		}
		if len(s) > 1 {
			return s, nil
		}
		// clients may send message containing only 0x0a byte to keep connection alive
		if r.OnKeepAlive != nil {
			if err := r.OnKeepAlive(); err != nil {
				return nil, err
			}
		}
	}
}
