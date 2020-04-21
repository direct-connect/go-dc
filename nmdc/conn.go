package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/direct-connect/go-dc/lineproto"
)

var (
	errWriterClosed = errors.New("nmdc: writer is closed")
)

const (
	SchemeNMDC  = "dchub" // URL scheme for NMDC protocol
	SchemeNMDCS = "nmdcs" // URL scheme for NMDC-over-TLS protocol
	DefaultPort = 411     // default port for client-hub connections
)

// ParseAddr parses an NMDC address as a URL. It will assume a dchub:// scheme if none is set.
func ParseAddr(addr string) (*url.URL, error) {
	if !strings.Contains(addr, "://") {
		addr = SchemeNMDC + "://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "nmdc" {
		u.Scheme = SchemeNMDC
	}
	if u.Scheme != SchemeNMDC && u.Scheme != SchemeNMDCS {
		return u, fmt.Errorf("unsupported protocol: %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}

// NormalizeAddr parses and normalizes the address to scheme://host[:port] format.
func NormalizeAddr(addr string) (string, error) {
	u, err := ParseAddr(addr)
	if err != nil {
		return "", err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		var err2 error
		host, _, err2 = net.SplitHostPort(u.Host + ":" + strconv.Itoa(DefaultPort))
		if err2 != nil {
			return "", err
		}
		err = nil
	}
	if host == "" {
		return "", fmt.Errorf("no hostname in address: %q", addr)
	}
	return u.String(), nil
}

type connConfig struct {
	NoZlib           bool
	Encoding         *Encoding
	EncodingCountry  *Encoding
	EncodingFallback *Encoding
	LineOpts         []lineproto.ConnOption
}

// ConnOption is an option that can be applied to NMDC connection.
type ConnOption interface {
	apply(c *connConfig)
}

type connOptionFunc func(c *connConfig)

func (f connOptionFunc) apply(c *connConfig) {
	f(c)
}

// WithEncoding sets protocol encoding options for the connection.
func WithEncoding(ec *Encoding) ConnOption {
	return connOptionFunc(func(c *connConfig) {
		c.Encoding = ec
	})
}

// WithNoZlib disables automatic zlib compression switch for the reader and
// forces an error when $ZOn command is received.
func WithNoZlib() ConnOption {
	return connOptionFunc(func(c *connConfig) {
		c.NoZlib = true
	})
}

// WithDomainName sets the domain name for the connection.
// It can later be used to get a clue about a fallback text encoding.
func WithDomainName(domain string) ConnOption {
	tld := domain
	if i := strings.LastIndexByte(tld, '.'); i >= 0 {
		tld = tld[i+1:]
	}
	// this only works when TLD = language code
	return WithLanguage(tld)
}

// WithLanguage sets the language code for the connection.
// It can later be used to get a clue about a fallback text encoding.
func WithLanguage(lang string) ConnOption {
	enc, ok := encodingForLanguage(lang)
	return connOptionFunc(func(c *connConfig) {
		if ok {
			ec := c.Encoding.Clone()
			ec.SetTextEncoding(enc)
			c.EncodingCountry = ec
		}
	})
}

// WithTextEncoding sets text encoding which will be used for non-UTF-8 strings.
func WithTextEncoding(enc TextEncoding) ConnOption {
	return connOptionFunc(func(c *connConfig) {
		ec := c.Encoding.Clone()
		ec.SetTextEncoding(enc)
		c.Encoding = ec
	})
}

// WithFallbackTextEncoding sets text encoding which will be used for non-UTF-8 strings.
func WithFallbackTextEncoding(enc TextEncoding) ConnOption {
	return connOptionFunc(func(c *connConfig) {
		ec := c.EncodingFallback.Clone()
		ec.SetTextEncoding(enc)
		c.EncodingFallback = ec
	})
}

// WithTextEncodingName is the same as WithTextEncoding, but accepts an encoding name.
func WithTextEncodingName(name string) (ConnOption, error) {
	enc, err := TextEncodingByName(name)
	if err != nil {
		return nil, err
	}
	return WithTextEncoding(enc), nil
}

// WithFallbackTextEncodingName is the same as WithTextEncoding, but accepts an encoding name.
func WithFallbackTextEncodingName(name string) (ConnOption, error) {
	enc, err := TextEncodingByName(name)
	if err != nil {
		return nil, err
	}
	return WithFallbackTextEncoding(enc), nil
}

// WithLineOpts can be used to apply lineproto.ConnOptions to the connection.
func WithLineOpts(opts ...lineproto.ConnOption) ConnOption {
	return connOptionFunc(func(c *connConfig) {
		c.LineOpts = append(c.LineOpts, opts...)
	})
}

func NewConn(conn net.Conn, opts ...ConnOption) *Conn {
	var conf connConfig
	conf.Encoding = DefaultEncoding()
	for _, opt := range opts {
		opt.apply(&conf)
	}
	lc := lineproto.NewConn(conn, Delimiter, conf.LineOpts...)
	c := &Conn{
		c:      lc,
		r:      NewReaderOn(lc),
		noZlib: conf.NoZlib,
	}
	c.r.SetEncoding(conf.Encoding)
	c.r.OnUnknownEncoding = func(text []byte) (TextEncoding, error) {
		for _, e := range []*Encoding{
			conf.EncodingCountry,
			conf.EncodingFallback,
		} {
			if e == nil {
				continue
			}
			if _, err := e.TextDecoder().Bytes(text); err == nil {
				return e.TextEncoding(), nil
			}
		}
		// replace invalid characters
		return nil, nil
	}
	c.r.OnRawMessage(c.onReaderZOn)
	return c
}

type Conn struct {
	c *lineproto.Conn
	r *Reader

	noZlib bool
	wpool  sync.Pool

	wmu    sync.Mutex
	onMsgW []func(m Message) (bool, error)
	w      *Writer
}

func (c *Conn) Encoding() *Encoding {
	return c.r.Encoding()
}

func (c *Conn) SetEncoding(ec *Encoding) {
	c.r.SetEncoding(ec)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.c.RemoteAddr()
}

func (c *Conn) Close() error {
	return c.c.Close()
}

func (c *Conn) CloseWith(m ...Message) error {
	ec := c.r.Encoding()
	// TODO: reuse write buffer
	buf := bytes.NewBuffer(nil)
	for _, m := range m {
		if err := ec.MarshalTo(buf, m); err != nil {
			_ = c.c.Close()
			return err
		}
	}
	return c.c.CloseWith(buf.Bytes())
}

func (c *Conn) CloseWithChatError(err error) error {
	return c.CloseWith(&ChatMessage{Text: err.Error()})
}

func (c *Conn) CloseWithRedirect(addr string) error {
	return c.CloseWith(&ForceMove{Address: addr})
}

func (c *Conn) Drop() {
	c.c.Drop()
}

func (c *Conn) onReaderZOn(cmd, args []byte) (bool, error) {
	if !bytes.Equal(cmd, []byte(zonName)) {
		return true, nil
	}
	if c.noZlib {
		return false, errors.New("nmdc: zlib is not allowed")
	}
	err := c.c.EnableZlibR()
	return false, err
}

func (c *Conn) ZOn(lvl int) error {
	return c.c.EnableZlibW([]byte(zonLine), lvl)
}

func (c *Conn) OnUnmarshalError(fnc func(line []byte, err error) (bool, error)) {
	c.r.OnUnmarshalError = fnc
}

func (c *Conn) OnLineR(fnc func(line []byte) (bool, error)) {
	c.c.OnLineR(fnc)
}

func (c *Conn) OnLineW(fnc func(line []byte) (bool, error)) {
	c.c.OnLineW(fnc)
}

func (c *Conn) OnRawMessageR(fnc func(cmd, data []byte) (bool, error)) {
	c.r.OnRawMessage(fnc)
}

func (c *Conn) OnMessageR(fnc func(m Message) (bool, error)) {
	c.r.OnMessage(fnc)
}

func (c *Conn) OnMessageW(fnc func(m Message) (bool, error)) {
	c.wmu.Lock()
	c.onMsgW = append(c.onMsgW, fnc)
	c.wmu.Unlock()
}

func (c *Conn) SetReadDeadline(deadline time.Time) error {
	return c.c.SetReadDeadline(deadline)
}

func (c *Conn) ReadMessage() (Message, error) {
	return c.r.ReadMsg()
}

func (c *Conn) ReadMessageTo(m Message) error {
	return c.r.ReadMsgTo(m)
}

func (c *Conn) ReadMessageToAny(m ...Message) (Message, error) {
	return c.r.ReadMsgToAny(m...)
}

func (c *Conn) newWriterOn(lw lineproto.LineWriter) *Writer {
	c.wmu.Lock()
	w := c.w
	if w != nil {
		c.w = nil
	}
	hooks := c.onMsgW
	c.wmu.Unlock()
	if w != nil {
		w.Reset(lw)
		return w
	}
	w, _ = c.wpool.Get().(*Writer)
	if w != nil {
		w.Reset(lw)
		return w
	}
	w = NewWriterOn(lw)
	for _, h := range hooks {
		w.OnMessage(h)
	}
	w.SetEncoding(c.r.Encoding())
	return w
}

func (c *Conn) putWriter(w *Writer) {
	w.Reset(nil)
	c.wmu.Lock()
	if c.w == nil {
		c.w = w
		c.wmu.Unlock()
		return
	}
	c.wmu.Unlock()
	c.wpool.Put(w)
}

func (c *Conn) BeginWrite() (*BatchWriter, error) {
	lw, err := c.c.BeginWrite()
	if err != nil {
		return nil, err
	}
	w := c.newWriterOn(lw)
	return &BatchWriter{c: c, lw: lw, w: w}, nil
}

type BatchWriter struct {
	c  *Conn
	lw *lineproto.BatchWriter
	w  *Writer
}

func (w *BatchWriter) SetWriteDeadline(deadline time.Time) error {
	return w.lw.SetWriteDeadline(deadline)
}

func (w *BatchWriter) WriteMsg(m ...Message) error {
	if w.w == nil {
		return errWriterClosed
	}
	return w.w.WriteMsg(m...)
}

func (w *BatchWriter) WriteMsgNow(m ...Message) error {
	if err := w.WriteMsg(m...); err != nil {
		return err
	}
	return w.Flush()
}

func (w *BatchWriter) WriteMsgDeadline(deadline time.Time, m ...Message) error {
	if err := w.SetWriteDeadline(deadline); err != nil {
		return err
	}
	return w.WriteMsgNow(m...)
}

func (w *BatchWriter) WriteLine(data []byte) error {
	return w.lw.WriteLine(data)
}

func (w *BatchWriter) Flush() error {
	return w.lw.Flush()
}

func (w *BatchWriter) Close() error {
	if w.w == nil {
		return errWriterClosed
	}
	err := w.lw.Close()
	nw := w.w
	w.w = nil
	w.c.putWriter(nw)
	return err
}

func (w *BatchWriter) Drop() {
	if w.w == nil {
		return
	}
	w.lw.Drop()
	nw := w.w
	w.w = nil
	w.c.putWriter(nw)
}

func (c *Conn) WriteLine(line []byte) error {
	w, err := c.BeginWrite()
	if err != nil {
		return err
	}
	if err = w.WriteLine(line); err != nil {
		w.Drop()
		return err
	}
	return w.Close()
}

func (c *Conn) WriteMessage(m ...Message) error {
	w, err := c.BeginWrite()
	if err != nil {
		return err
	}
	if err = w.WriteMsg(m...); err != nil {
		w.Drop()
		return err
	}
	return w.Close()
}

func (c *Conn) WriteLineAsync(line []byte) error {
	w, err := c.BeginWriteAsync()
	if err != nil {
		return err
	}
	defer w.Close()
	if err = w.WriteLine(line); err != nil {
		return err
	}
	return w.Close()
}

func (c *Conn) WriteMessageAsync(m ...Message) error {
	w, err := c.BeginWriteAsync()
	if err != nil {
		return err
	}
	defer w.Close()
	if err = w.WriteMsg(m...); err != nil {
		return err
	}
	return w.Close()
}

func (c *Conn) BeginWriteAsync() (*AsyncWriter, error) {
	lw, err := c.c.BeginWriteAsync()
	if err != nil {
		return nil, err
	}
	w := c.newWriterOn(lw)
	return &AsyncWriter{c: c, lw: lw, w: w}, nil
}

type AsyncWriter struct {
	c  *Conn
	lw *lineproto.AsyncWriter
	w  *Writer
}

func (w *AsyncWriter) WriteMsg(m ...Message) error {
	if w.w == nil {
		return errWriterClosed
	}
	return w.w.WriteMsg(m...)
}

func (w *AsyncWriter) WriteLine(data []byte) error {
	return w.lw.WriteLine(data)
}

func (w *AsyncWriter) Flush() error {
	return w.lw.Flush()
}

func (w *AsyncWriter) Close() error {
	if w.w == nil {
		return errWriterClosed
	}
	err := w.lw.Close()
	nw := w.w
	w.w = nil
	w.c.putWriter(nw)
	return err
}
