package adc

import (
	"bytes"
	"fmt"
	"github.com/direct-connect/go-dc/adc/types"
	"github.com/direct-connect/go-dc/tiger"
	"strings"
)

var (
	_ Marshaler   = SID{}
	_ Unmarshaler = (*SID)(nil)
)

type SID = types.SID

var (
	_ Marshaler   = CID{}
	_ Unmarshaler = (*CID)(nil)
)

type CID = types.CID

var (
	_ Marshaler   = PID{}
	_ Unmarshaler = (*PID)(nil)
)

type PID = CID

var (
	_ Marshaler   = TTH{}
	_ Unmarshaler = (*TTH)(nil)
)

// TTH is a Tiger Tree Hash value.
type TTH = tiger.Hash

type String string

func (f String) MarshalADC(buf *bytes.Buffer) error {
	buf.Write(escape(string(f)))
	return nil
}

func (f *String) UnmarshalADC(s []byte) error {
	*f = String(unescape(s))
	return nil
}

type Error struct {
	Status
}

func (e Error) Error() string {
	return fmt.Sprintf("code %d: %s", e.Code, e.Msg)
}

type AddFeatures []string

func (f AddFeatures) MarshalADC(buf *bytes.Buffer) error {
	for i, sf := range f {
		if i > 0 {
			buf.WriteString(" AD" + sf)
		} else {
			buf.WriteString("AD" + sf)
		}
	}
	return nil
}

type Feature [4]byte

func (f Feature) String() string {
	return string(f[:])
}

func (f Feature) MarshalADC(buf *bytes.Buffer) error {
	buf.Write(f[:])
	return nil
}

func (f *Feature) UnmarshalADC(s []byte) error {
	if len(s) != 4 {
		return fmt.Errorf("malformed feature [%d]", len(s))
	}
	var v Feature
	copy(v[:], s)
	*f = v
	return nil
}

type ModFeatures map[Feature]bool

func (f ModFeatures) Clone() ModFeatures {
	mf := make(ModFeatures, len(f))
	for k, v := range f {
		mf[k] = v
	}
	return mf
}

func (f ModFeatures) MarshalADC(buf *bytes.Buffer) error {
	first := true
	for fea, st := range f {
		if !first {
			buf.WriteRune(' ')
		} else {
			first = false
		}
		if st {
			buf.WriteString("AD")
		} else {
			buf.WriteString("RM")
		}
		buf.WriteString(fea.String())
	}
	return nil
}

func (f *ModFeatures) UnmarshalADC(s []byte) error {
	// TODO: will parse strings of any length; should limit to 4 bytes
	// TODO: use bytes or Feature in slices
	var out struct {
		Add []string `adc:"AD"`
		Rm  []string `adc:"RM"`
	}
	if err := Unmarshal(s, &out); err != nil {
		return err
	}
	m := *f
	if m == nil {
		m = make(ModFeatures)
		*f = m
	}
	for _, name := range out.Rm {
		var fea Feature
		if err := fea.UnmarshalADC([]byte(name)); err != nil {
			return err
		}
		m[fea] = false
	}
	for _, name := range out.Add {
		var fea Feature
		if err := fea.UnmarshalADC([]byte(name)); err != nil {
			return err
		}
		m[fea] = true
	}
	return nil
}

func (f ModFeatures) IsSet(s Feature) bool {
	if f == nil {
		return false
	}
	_, ok := f[s]
	return ok
}

func (f ModFeatures) SetFrom(fp ModFeatures) ModFeatures {
	if f == nil && fp == nil {
		return nil
	}
	fi := f.Clone()
	for name, add := range fp {
		fi[name] = add
	}
	return fi
}

func (f ModFeatures) Intersect(fp ModFeatures) ModFeatures {
	if f == nil || fp == nil {
		return nil
	}
	fi := make(ModFeatures)
	for name, add := range f {
		if add {
			if sup, ok := fp[name]; sup && ok {
				fi[name] = true
			}
		}
	}
	return fi
}

func (f ModFeatures) Join() string {
	var arr []string
	for name, add := range f {
		if add {
			arr = append(arr, name.String())
		}
	}
	return strings.Join(arr, ",")
}

var (
	_ Marshaler   = (ExtFeatures)(nil)
	_ Unmarshaler = (*ExtFeatures)(nil)
)

type ExtFeatures []Feature

func (f ExtFeatures) Has(s Feature) bool {
	for _, sf := range f {
		if sf == s {
			return true
		}
	}
	return false
}

func (f ExtFeatures) MarshalADC(buf *bytes.Buffer) error {
	for i, fea := range f {
		if i != 0 {
			buf.WriteByte(',')
		}
		err := fea.MarshalADC(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *ExtFeatures) UnmarshalADC(s []byte) error {
	if len(s) < 1 {
		return nil
	}
	sub := bytes.Split(s, []byte(","))
	arr := make(ExtFeatures, 0, len(sub))
	for _, s := range sub {
		var fea Feature
		if err := fea.UnmarshalADC(s); err != nil {
			return err
		}
		arr = append(arr, fea)
	}
	*f = arr
	return nil
}

type BoolInt bool

func (f BoolInt) MarshalADC(buf *bytes.Buffer) error {
	if bool(f) {
		buf.WriteByte('1')
		return nil
	}
	buf.WriteByte('0')
	return nil
}

func (f *BoolInt) UnmarshalADC(s []byte) error {
	if len(s) != 1 {
		return fmt.Errorf("wrong bool value: '%s'", s)
	}
	switch s[0] {
	case '0':
		*f = false
	case '1':
		*f = true
	default:
		return fmt.Errorf("wrong bool value: '%s'", s)
	}
	return nil
}

var (
	_ Marshaler   = Path{}
	_ Unmarshaler = (*Path)(nil)
)

type Path []string

func (p Path) MarshalADC(buf *bytes.Buffer) error {
	for i, a := range p {
		if i != 0 {
			buf.WriteByte('/')
		}
		err := String(a).MarshalADC(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Path) UnmarshalADC(data []byte) error {
	arr := bytes.Split(data, []byte("/"))
	for _, a := range arr {
		var path String
		if err := path.UnmarshalADC(a); err != nil {
			return err
		}
		*p = append(*p, string(path))
	}
	return nil
}
