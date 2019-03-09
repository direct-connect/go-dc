package nmdc

import (
	"bytes"
)

// DefaultKeyMagic is a magic byte used in the C-H and C-C handshakes.
const DefaultKeyMagic = 5

var keyReplace = map[byte]string{
	0:   "/%DCN000%/",
	5:   "/%DCN005%/",
	36:  "/%DCN036%/",
	96:  "/%DCN096%/",
	124: "/%DCN124%/",
	126: "/%DCN126%/",
}

func init() {
	RegisterMessage(&Lock{})
	RegisterMessage(&Key{})
}

// Lock is a pseudo-cryptographic challenge sent by the server to the client.
//
// http://nmdc.sourceforge.net/NMDC.html#_lock
type Lock struct {
	NoExt bool
	Lock  string
	PK    string
	Ref   string
}

func (*Lock) Type() string {
	return "Lock"
}

func (m *Lock) MarshalNMDC(_ *TextEncoder, buf *bytes.Buffer) error {
	if !m.NoExt {
		buf.WriteString(extLockPref)
	}
	buf.WriteString(m.Lock)
	if m.PK != "" {
		buf.WriteString(" Pk=")
		buf.WriteString(m.PK)
	}
	if m.Ref != "" {
		if m.PK == "" {
			buf.WriteString(" ")
		}
		buf.WriteString("Ref=")
		buf.WriteString(m.Ref)
	}
	return nil
}

func (m *Lock) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	*m = Lock{NoExt: true}
	if bytes.HasPrefix(data, []byte(extLockPref)) {
		m.NoExt = false
		data = data[len(extLockPref):]
	}
	i := bytes.Index(data, []byte(" "))
	if i < 0 {
		m.Lock = string(data)
		return nil
	}
	m.Lock = string(data[:i])

	data = data[i+1:]
	if bytes.HasPrefix(data, []byte("Pk=")) {
		data = bytes.TrimPrefix(data, []byte("Pk="))
	}
	i = bytes.Index(data, []byte("Ref="))
	if i >= 0 {
		m.PK = string(data[:i])
		m.Ref = string(data[i+4:])
	} else {
		m.PK = string(data)
	}
	return nil
}

func (m *Lock) LockString(full bool) string {
	if !full {
		if m.NoExt {
			return m.Lock
		}
		return extLockPref + m.Lock
	}
	buf := bytes.NewBuffer(nil)
	err := m.MarshalNMDC(nil, buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// Key calculates a response to the challenge.
func (m *Lock) Key() *Key {
	return m.CustomKey(DefaultKeyMagic, false)
}

// CustomKey calculates a response to the challenge and allows to specify additional parameters.
func (m *Lock) CustomKey(magic byte, full bool) *Key {
	lock := []byte(m.LockString(full))

	n := len(lock)
	key := make([]byte, n)

	key[0] = byte((lock[0] ^ lock[n-1] ^ lock[n-2] ^ magic) & 0xFF)
	for i := 1; i < n; i++ {
		key[i] = (lock[i] ^ lock[i-1]) & 0xFF
	}
	for i := 0; i < n; i++ {
		// swap nibbles
		key[i] = byte((((key[i] << 4) & 0xF0) | ((key[i] >> 4) & 0x0F)) & 0xFF)
	}
	buf := bytes.NewBuffer(nil)
	buf.Grow(len(key))
	for _, v := range key {
		if esc, ok := keyReplace[v]; ok {
			buf.WriteString(esc)
		} else {
			buf.WriteByte(v)
		}
	}
	return &Key{Key: buf.String()}
}

// Key is a response to a pseudo-cryptographic challenge represented by Lock.
//
// http://nmdc.sourceforge.net/NMDC.html#_key
type Key struct {
	Key string
}

func (*Key) Type() string {
	return "Key"
}

func (m *Key) MarshalNMDC(_ *TextEncoder, buf *bytes.Buffer) error {
	buf.WriteString(m.Key)
	return nil
}

func (m *Key) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	m.Key = string(data)
	return nil
}
