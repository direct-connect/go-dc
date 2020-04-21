package nmdc

import (
	"io"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAddr(t *testing.T) {
	var cases = []struct {
		name string
		addr string
		exp  string
		err  bool
	}{
		{
			name: "only host",
			addr: "localhost",
			exp:  SchemeNMDC + "://localhost",
		},
		{
			name: "only host ipv6",
			addr: "[::]",
			exp:  SchemeNMDC + "://[::]",
		},
		{
			name: "only port",
			addr: ":411",
			err:  true,
		},
		{
			name: "scheme and host",
			addr: "dchub://localhost",
			exp:  SchemeNMDC + "://localhost",
		},
		{
			name: "scheme and host ipv6",
			addr: "dchub://[::]",
			exp:  SchemeNMDC + "://[::]",
		},
		{
			name: "host and port",
			addr: "dchub://localhost:411",
			exp:  SchemeNMDC + "://localhost:411",
		},
		{
			name: "host and port ipv6",
			addr: "dchub://[::]:411",
			exp:  SchemeNMDC + "://[::]:411",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeAddr(c.addr)
			if c.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.exp, got)
		})
	}
}

func connWithData(t testing.TB, data string) (net.Conn, func()) {
	cr, cw := net.Pipe()
	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		defer cw.Close()
		_, err := cw.Write([]byte(data))
		if err != nil {
			errc <- err
		}
	}()
	return cr, func() {
		cw.Close()
		cr.Close()
		require.NoError(t, <-errc)
	}
}

func chatText(text string) *ChatMessage {
	return &ChatMessage{Text: text}
}

func encText(text, enc string) string {
	e, err := TextEncodingByName(enc)
	if err != nil {
		panic(err)
	}
	text, err = e.NewEncoder().String(text)
	if err != nil {
		panic(err)
	}
	return text
}

func TestSwitchEncoding(t *testing.T) {
	require.Nil(t, DefaultEncoding().TextEncoding())
	defer func() {
		require.Nil(t, DefaultEncoding().TextEncoding())
	}()
	const (
		encSE   = "cp1252"
		encRU   = "cp1251"
		encRU2  = "koi8-r"
		helloEN = "hello world"
		helloSE = "hej världen"
		helloRU = "привет мир"
		helloJA = "こんにちは世界"
	)
	var (
		helloSEEnc  = encText(helloSE, encSE)
		helloRUEnc  = encText(helloRU, encRU)
		helloRUEnc2 = encText(helloRU, encRU2)
	)
	joinCmds := func(lines ...string) string {
		lines = append(lines[:len(lines):len(lines)], "")
		return strings.Join(lines, string(Delimiter))
	}
	mustWithFallbackEncodingName := func(name string) ConnOption {
		opt, err := WithFallbackTextEncodingName(name)
		if err != nil {
			panic(err)
		}
		return opt
	}
	var cases = []struct {
		name string
		data string
		opts []ConnOption
		exp  []Message
	}{
		{
			name: "utf8",
			data: joinCmds(
				helloEN,
				helloSE,
				helloRU,
				helloJA,
			),
			exp: []Message{
				chatText(helloEN),
				chatText(helloSE),
				chatText(helloRU),
				chatText(helloJA),
			},
		},
		{
			name: "mask and utf8",
			data: joinCmds(
				helloEN,    // ASCII
				helloSEEnc, // CP1252
				helloRUEnc, // CP1251
				helloJA,    // UTF-8
			),
			exp: []Message{
				chatText(helloEN),
				// non-ASCII and non-UTF-8 are masked
				chatText("hej v�rlden"),
				chatText("������ ���"),
				chatText(helloJA),
			},
		},
		{
			name: "fallback",
			data: joinCmds(
				helloEN,    // ASCII
				helloSEEnc, // CP1252
				helloEN,    // ASCII
				helloJA,    // UTF-8
				helloSEEnc, // CP1252 still in use
			),
			opts: []ConnOption{
				mustWithFallbackEncodingName(encSE),
			},
			exp: []Message{
				chatText(helloEN),
				chatText(helloSE),
				chatText(helloEN),
				chatText(helloJA),
				chatText(helloSE),
			},
		},
		{
			name: "fallback 2",
			data: joinCmds(
				helloEN,    // ASCII
				helloRUEnc, // CP1251
				helloEN,    // ASCII
				helloJA,    // UTF-8
				helloRUEnc, // CP1251 still in use
			),
			opts: []ConnOption{
				mustWithFallbackEncodingName(encRU),
			},
			exp: []Message{
				chatText(helloEN),
				chatText(helloRU),
				chatText(helloEN),
				chatText(helloJA),
				chatText(helloRU),
			},
		},
		{
			name: "tld",
			data: joinCmds(
				helloEN,    // ASCII
				helloRUEnc, // CP1251, which is the default encoding for RU
				helloEN,    // ASCII
				helloJA,    // UTF-8
				helloRUEnc, // CP1251 still in use
			),
			opts: []ConnOption{
				WithDomainName("example.ru"),
			},
			exp: []Message{
				chatText(helloEN),
				chatText(helloRU),
				chatText(helloEN),
				chatText(helloJA),
				chatText(helloRU),
			},
		},
		{
			name: "tld and fallback",
			data: joinCmds(
				helloEN,    // ASCII
				helloSEEnc, // CP1252, which is both the fallback and the default
				helloEN,    // ASCII
				helloJA,    // UTF-8
				helloSEEnc, // CP1252 still in use
			),
			opts: []ConnOption{
				WithDomainName("example.com"),
				mustWithFallbackEncodingName(encSE),
			},
			exp: []Message{
				chatText(helloEN),
				chatText(helloSE),
				chatText(helloEN),
				chatText(helloJA),
				chatText(helloSE),
			},
		},
		{
			name: "different tld and fallback",
			data: joinCmds(
				helloEN,     // ASCII
				helloRUEnc,  // CP1251 (default for TLD)
				helloRUEnc2, // KOI8-R (fallback)
				helloJA,     // UTF-8
			),
			opts: []ConnOption{
				WithDomainName("example.ru"),
				mustWithFallbackEncodingName(encRU2),
			},
			exp: []Message{
				chatText(helloEN),
				chatText(helloRU),      // CP1251 selected
				chatText("РТЙЧЕФ НЙТ"), // CP1251 still in use
				chatText(helloJA),
			},
		},
		{
			name: "different tld and fallback 2",
			data: joinCmds(
				helloEN,     // ASCII
				helloRUEnc2, // KOI8-R (fallback)
				helloRUEnc,  // CP1251 (default for TLD)
				helloJA,     // UTF-8
			),
			opts: []ConnOption{
				WithDomainName("example.ru"),
				mustWithFallbackEncodingName(encRU2),
			},
			exp: []Message{
				chatText(helloEN),
				chatText("РТЙЧЕФ НЙТ"), // TODO: KOI8-R should be accepted instead of CP1251
				chatText(helloRU),      // CP1251 selected previously
				chatText(helloJA),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conn, closer := connWithData(t, c.data)
			defer closer()

			nc := NewConn(conn, c.opts...)
			var got []Message
			for {
				m, err := nc.ReadMessage()
				if err == io.EOF || err == io.ErrClosedPipe {
					break
				}
				require.NoError(t, err)
				got = append(got, m)
			}
			require.Nil(t, DefaultEncoding().TextEncoding())
			require.Equal(t, c.exp, got)
		})
	}
}
