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

const (
	invalidCharsName = "$\x00\r\n\t"
)

var legacyUnescaper = strings.NewReplacer(
	"/%DCN000%/", "\x00",
	"/%DCN005%/", "\x05",
	"/%DCN036%/", "$",
	"/%DCN096%/", "`",
	"/%DCN124%/", "|",
	"/%DCN126%/", "~",
)

var htmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"$", "&#36;",
	"|", "&#124;",
)

var htmlEscaperName = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"$", "&#36;",
	"|", "&#124;",
)

// Escape the string value.
func Escape(s string) string {
	return htmlEscaper.Replace(s)
}

// EscapeName escapes the string value according to name escaping rules.
func EscapeName(s string) string {
	return htmlEscaperName.Replace(s)
}

// Unescape string value.
func Unescape(s string) string {
	if strings.Contains(s, "/%DCN") {
		s = legacyUnescaper.Replace(s)
	}
	s = html.UnescapeString(s)
	return s
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
	} else if strings.ContainsAny(string(s), invalidCharsName) {
		return fmt.Errorf("invalid characters in name: %q", string(s))
	}
	str := string(s)
	if enc != nil {
		var err error
		str, err = enc.String(str)
		if err != nil {
			return err
		}
	}
	str = EscapeName(str)
	buf.WriteString(str)
	return nil
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
	if strings.Contains(string(s), "\x00") {
		return errors.New("invalid characters in text")
	}
	str := string(s)
	if enc != nil {
		var err error
		str, err = enc.String(str)
		if err != nil {
			return err
		}
	}
	str = Escape(str)
	buf.WriteString(str)
	return nil
}

func (s *String) UnmarshalNMDC(dec *TextDecoder, data []byte) error {
	if bytes.Contains(data, []byte{0x00}) {
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
