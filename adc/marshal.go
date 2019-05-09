package adc

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/direct-connect/go-dc/tiger"
)

var (
	_ Unmarshaler = (*tiger.Hash)(nil)
	_ Marshaler   = tiger.Hash{}
)

var (
	escaper   = strings.NewReplacer(`\`, `\\`, " ", `\s`, "\n", `\n`)
	unescaper = strings.NewReplacer(`\s`, " ", `\n`, "\n", `\\`, `\`)
)

func escape(s string) []byte {
	return []byte(escaper.Replace(s))
}

func unescape(s []byte) string {
	return unescaper.Replace(string(s))
}

// Marshaler is an interface for ADC messages that can marshal itself.
type Marshaler interface {
	MarshalADC(buf *bytes.Buffer) error
}

// Unmarshaler is an interface for ADC messages that can marshal itself.
type Unmarshaler interface {
	UnmarshalADC(data []byte) error
}

func unmarshalValue(s []byte, rv reflect.Value) error {
	switch fv := rv.Addr().Interface().(type) {
	case Unmarshaler:
		return fv.UnmarshalADC(s)
	}
	if len(s) == 0 {
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16:
		bits := 64
		switch rv.Kind() {
		case reflect.Int32, reflect.Int16:
			bits = 32
		}
		sv := string(s)
		vi, err := strconv.ParseInt(sv, 10, bits)
		if err != nil {
			vf, err2 := strconv.ParseFloat(sv, bits)
			if err2 != nil {
				return err
			} else if math.Round(vf) != vf {
				return err2
			}
			vi = int64(vf)
		}
		rv.SetInt(vi)
		return nil
	case reflect.String:
		rv.SetString(unescape(s))
		return nil
	case reflect.Ptr:
		if len(s) == 0 {
			return nil
		}
		nv := reflect.New(rv.Type().Elem())
		err := unmarshalValue(s, nv.Elem())
		if err == nil {
			rv.Set(nv)
		}
		return err
	case reflect.Bool:
		if len(s) == 0 {
			rv.SetBool(false)
			return nil
		} else if len(s) != 1 {
			return errors.New("invalid bool value: " + string(s))
		}
		rv.SetBool(s[0] != 0)
		return nil
	}
	return fmt.Errorf("unknown type: %v", rv.Type())
}

// Unmarshal decodes ADC message to a given type.
func Unmarshal(s []byte, o interface{}) error {
	if m, ok := o.(Unmarshaler); ok {
		return m.UnmarshalADC(s)
	}
	rv := reflect.ValueOf(o)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("pointer expected, got: %T", o)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("struct expected, got: %T", o)
	}
	rt := rv.Type()

	sub := bytes.Split(s, []byte(" "))
	for i := 0; i < rt.NumField(); i++ {
		fld := rt.Field(i)
		tag := strings.SplitN(fld.Tag.Get(`adc`), ",", 2)[0]
		if tag == "" || tag == "-" {
			continue
		} else if tag == "#" {
			v := sub[0]
			sub = sub[1:]
			if err := unmarshalValue(v, rv.Field(i)); err != nil {
				return fmt.Errorf("error on field %s: %s", fld.Name, err)
			}
			continue
		}
		btag := []byte(tag)
		var vals [][]byte
		for _, ss := range sub {
			if bytes.HasPrefix(ss, btag) {
				vals = append(vals, bytes.TrimPrefix(ss, btag))
			}
		}
		if len(vals) > 0 {
			if m, ok := rv.Field(i).Addr().Interface().(Unmarshaler); ok {
				if err := m.UnmarshalADC(vals[0]); err != nil {
					return fmt.Errorf("error on field %s: %s", fld.Name, err)
				}
			} else if fld.Type.Kind() == reflect.Slice {
				nv := reflect.MakeSlice(fld.Type, len(vals), len(vals))
				for j, sv := range vals {
					if err := unmarshalValue(sv, nv.Index(j)); err != nil {
						return fmt.Errorf("error on field %s: %s", fld.Name, err)
					}
				}
				rv.Field(i).Set(nv)
			} else {
				if len(vals) > 1 {
					return fmt.Errorf("error on field %s: expected single value", fld.Name)
				} else {
					if err := unmarshalValue(vals[0], rv.Field(i)); err != nil {
						return fmt.Errorf("error on field %s: %s", fld.Name, err)
					}
				}
			}
		}
	}
	return nil
}

func marshalValue(buf *bytes.Buffer, o interface{}) error {
	switch v := o.(type) {
	case Marshaler:
		return v.MarshalADC(buf)
	}
	rv := reflect.ValueOf(o)
	switch rv.Kind() {
	case reflect.String:
		s := rv.String()
		buf.Write(escape(s))
		return nil
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16:
		v := rv.Int()
		buf.WriteString(strconv.FormatInt(v, 10))
		return nil
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16:
		v := rv.Uint()
		buf.WriteString(strconv.FormatUint(v, 10))
		return nil
	case reflect.Bool:
		v := rv.Bool()
		if v {
			buf.WriteByte('1')
			return nil
		}
		buf.WriteByte('0')
		return nil
	case reflect.Ptr:
		if rv.IsNil() {
			return nil
		}
		return marshalValue(buf, rv.Elem().Interface())
	}
	return fmt.Errorf("unsupported type: %T", o)
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return v.Interface() == reflect.Zero(v.Type()).Interface()
	}
}

// Marshal encodes ADC message payload to a buffer. It won't encode the message name.
func Marshal(buf *bytes.Buffer, o Message) error {
	if o == nil {
		return nil
	}
	if m, ok := o.(Marshaler); ok {
		return m.MarshalADC(buf)
	}
	rv := reflect.ValueOf(o)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("struct expected, got: %T", o)
	}
	rt := rv.Type()
	first := true
	writeTag := func(tag string) {
		if !first {
			buf.WriteRune(' ')
		} else {
			first = false
		}
		if tag != `#` {
			buf.WriteString(tag)
		}
	}
	for i := 0; i < rt.NumField(); i++ {
		fld := rt.Field(i)
		tag := fld.Tag.Get(`adc`)
		if tag == "" || tag == "-" {
			continue
		}
		sub := strings.SplitN(tag, ",", 2)
		tag = sub[0]
		omit := tag != "#"
		if len(sub) > 1 && sub[1] == "req" {
			omit = false
		}
		if omit && isZero(rv.Field(i)) {
			continue
		}
		if m, ok := rv.Field(i).Interface().(Marshaler); ok {
			writeTag(tag)
			if err := m.MarshalADC(buf); err != nil {
				return fmt.Errorf("cannot marshal field %s: %v", fld.Name, err)
			}
			continue
		}
		if tag != "#" && fld.Type.Kind() == reflect.Slice {
			for j := 0; j < rv.Field(i).Len(); j++ {
				writeTag(tag)
				err := marshalValue(buf, rv.Field(i).Index(j).Interface())
				if err != nil {
					return fmt.Errorf("cannot marshal field %s: %v", fld.Name, err)
				}
			}
		} else {
			writeTag(tag)
			err := marshalValue(buf, rv.Field(i).Interface())
			if err != nil {
				return fmt.Errorf("cannot marshal field %s: %v", fld.Name, err)
			}
		}
	}
	return nil
}

// MustMarshal encodes an ADC message payload into a byte slice and panics on error.
func MustMarshal(o Message) []byte {
	buf := bytes.NewBuffer(nil)
	err := Marshal(buf, o)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}
