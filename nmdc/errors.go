package nmdc

import (
	"bytes"
	"errors"
)

func init() {
	RegisterMessage(&Failed{})
	RegisterMessage(&Error{})
}

type Failed struct {
	Err error
}

func (m *Failed) Type() string {
	return "Failed"
}

func (m *Failed) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if m.Err == nil {
		return nil
	}
	if err := String(m.Err.Error()).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	return nil
}

func (m *Failed) UnmarshalNMDC(dec *TextDecoder, text []byte) error {
	var str String
	if err := str.UnmarshalNMDC(dec, text); err != nil {
		return err
	}
	if str != "" {
		m.Err = errors.New(string(str))
	}
	return nil
}

type Error struct {
	Err error
}

func (m *Error) Type() string {
	return "Error"
}

func (m *Error) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if m.Err == nil {
		return nil
	}
	if err := String(m.Err.Error()).MarshalNMDC(enc, buf); err != nil {
		return err
	}
	return nil
}

func (m *Error) UnmarshalNMDC(dec *TextDecoder, text []byte) error {
	var str String
	if err := str.UnmarshalNMDC(dec, text); err != nil {
		return err
	}
	if str != "" {
		m.Err = errors.New(string(str))
	}
	return nil
}
