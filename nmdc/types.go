package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/direct-connect/go-dc/tiger"
)

var invalidCharsNameI [256]bool

const (
	invalidCharsName = "$\x00\r\n\t"
)

func init() {
	for _, c := range invalidCharsName {
		invalidCharsNameI[c] = true
	}
}

var legacyUnescaper = strings.NewReplacer(
	"/%DCN000%/", "\x00",
	"/%DCN005%/", "\x05",
	"/%DCN036%/", "$",
	"/%DCN096%/", "`",
	"/%DCN124%/", string(Delimiter),
	"/%DCN126%/", "~",
)

var escapeCharsString = [256]string{
	'&':       "&amp;",
	'$':       "&#36;",
	Delimiter: "&#124;",
}

var escapeCharsName = [256]string{
	'&':       "&amp;",
	'<':       "&lt;",
	'>':       "&gt;",
	'$':       "&#36;",
	Delimiter: "&#124;",
}

func UnescapeBytes(b []byte) []byte {
	h := bytes.IndexByte(b, '&') >= 0
	d := bytes.Contains(b, []byte("/%DCN"))
	if !h && !d {
		return b
	}
	s := string(b)
	if d {
		s = legacyUnescaper.Replace(s)
	}
	if h {
		s = html.UnescapeString(s)
	}
	return []byte(s)
}

type TTH = tiger.Hash

// NoArgs is an embeddable type for protocol commands with no arguments.
type NoArgs struct{}

func (*NoArgs) MarshalNMDC(_ *Encoding, _ *bytes.Buffer) error {
	return nil
}

func (*NoArgs) UnmarshalNMDC(_ *Encoding, data []byte) error {
	if len(data) != 0 {
		return errors.New("unexpected argument for the command")
	}
	return nil
}

// Name is a string encoded and decoded as a NMDC user name.
// It has more restrictions than a String type.
type Name string

func (s Name) MarshalNMDC(ec *Encoding, buf *bytes.Buffer) error {
	if len(s) > maxName {
		return errors.New("name is too long")
	}
	str := string(s)
	if enc := ec.TextEncoder(); enc != nil {
		var err error
		str, err = enc.String(str)
		if err != nil {
			return err
		}
	}
	return escapeName(buf, str)
}

func (s *Name) UnmarshalNMDC(ec *Encoding, data []byte) error {
	if ec.MaxUserName > 0 && len(data) > ec.MaxUserName {
		return errors.New("name is too long")
	}
	if ec.Raw {
		*s = Name(data)
		return nil
	}
	if bytes.ContainsAny(data, invalidCharsName) {
		return fmt.Errorf("invalid characters in name: %q", string(data))
	}
	data, err := ec.DecodeStringBytes(data)
	if err != nil {
		return err
	}
	*s = Name(data)
	return nil
}

// String is a value type encoded and decoded as a NMDC string value.
type String string

func (s String) MarshalNMDC(ec *Encoding, buf *bytes.Buffer) error {
	str := string(s)
	if enc := ec.TextEncoder(); enc != nil {
		var err error
		str, err = enc.String(str)
		if err != nil {
			return err
		}
	}
	return escapeString(buf, str)
}

func (s *String) UnmarshalNMDC(ec *Encoding, data []byte) error {
	if !ec.Raw && bytes.IndexByte(data, 0x00) >= 0 {
		return errors.New("invalid characters in text")
	}
	data, err := ec.DecodeStringBytes(data)
	if err != nil {
		return err
	}
	*s = String(data)
	return nil
}
