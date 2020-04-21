package nmdc

import "bytes"

type Buffer struct {
	buf   bytes.Buffer
	raw   []Message
	cache []encoded
}

type encoded struct {
	ec   *Encoding
	data []byte
}

// WriteMessage adds messages to the buffer. They must not be changed while buffer is in use.
func (b *Buffer) WriteMessage(m ...Message) error {
	if b.raw == nil {
		b.raw = m[:len(m):len(m)]
	} else {
		b.raw = append(b.raw, m...)
	}
	return nil
}

// BytesFor returns the serialization of the message buffer in specific encoding.
func (b *Buffer) BytesFor(ec *Encoding) ([]byte, error) {
	for _, c := range b.cache {
		if c.ec.Equal(ec) {
			return c.data, nil
		}
	}
	b.buf.Reset()
	for _, m := range b.raw {
		if err := ec.MarshalTo(&b.buf, m); err != nil {
			return nil, err
		}
	}
	data := append([]byte{}, b.buf.Bytes()...)
	b.cache = append(b.cache, encoded{ec: ec, data: data})
	return data, nil
}
