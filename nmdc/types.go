package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"strings"
	"unicode/utf8"

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
	"/%DCN124%/", string(lineDelim),
	"/%DCN126%/", "~",
)

var escapeCharsString = [256]string{
	'&':       "&amp;",
	'$':       "&#36;",
	lineDelim: "&#124;",
}

var escapeCharsName = [256]string{
	'&':       "&amp;",
	'<':       "&lt;",
	'>':       "&gt;",
	'$':       "&#36;",
	lineDelim: "&#124;",
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

func (*NoArgs) MarshalNMDC(_ *TextEncoder, _ *bytes.Buffer) error {
	return nil
}

func (*NoArgs) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	if len(data) != 0 {
		return errors.New("unexpected argument for the command")
	}
	return nil
}

// Name is a string encoded and decoded as a NMDC user name.
// It has more restrictions than a String type.
type Name string

func (s Name) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	if len(s) > maxName {
		return errors.New("name is too long")
	}
	str := string(s)
	if enc != nil {
		var err error
		str, err = enc.String(str)
		if err != nil {
			return err
		}
	}
	return escapeName(buf, str)
}

func (s *Name) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	if len(data) > maxName {
		return errors.New("name is too long")
	} else if bytes.ContainsAny(data, invalidCharsName) {
		return fmt.Errorf("invalid characters in name: %q", string(data))
	}
	data = UnescapeBytes(data)
	if dec != nil {
		var err error
		data, err = dec.Bytes(data)
		if err != nil {
			return err
		}
	}
	if !utf8.Valid(data) {
		return &errUnknownEncoding{text: data}
	}
	*s = Name(data)
	return nil
}

// String is a value type encoded and decoded as a NMDC string value.
type String string

func (s String) MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error {
	str := string(s)
	if enc != nil {
		var err error
		str, err = enc.String(str)
		if err != nil {
			return err
		}
	}
	return escapeString(buf, str)
}

func (s *String) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	if bytes.IndexByte(data, 0x00) >= 0 {
		return errors.New("invalid characters in text")
	}
	data = UnescapeBytes(data)
	if dec != nil {
		var err error
		data, err = dec.Bytes(data)
		if err != nil {
			return err
		}
	}
	if !utf8.Valid(data) {
		return &errUnknownEncoding{text: data}
	}
	*s = String(data)
	return nil
}
