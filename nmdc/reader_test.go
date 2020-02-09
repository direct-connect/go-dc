package nmdc

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var casesReader = []struct {
	name  string
	input string
	exp   []Message
	err   error
}{
	{
		name:  "eof",
		input: ``,
		exp:   nil,
	},
	{
		name:  "pings",
		input: `|||`,
		exp:   nil,
	},
	{
		name:  "empty cmd",
		input: `$|`,
		exp:   nil,
		err: &ErrProtocolViolation{
			Err: errors.New("command name is empty"),
		},
	},
	{
		name:  "GetNickList",
		input: `$GetNickList|`,
		exp: []Message{
			&GetNickList{},
		},
	},
	{
		name:  "null char in command",
		input: "$SomeCommand\x00|",
		err: &ErrProtocolViolation{
			Err: errors.New("message should not contain null characters"),
		},
	},
	{
		name:  "non-ascii in command",
		input: "$Some\tCommand|",
		err: &ErrProtocolViolation{
			Err: errors.New(`command name should be in acsii: "Some\tCommand"`),
		},
	},
	{
		name:  "to",
		input: "$To: alice From: bob $<bob> hi|",
		exp: []Message{
			&PrivateMessage{
				To:   "alice",
				From: "bob",
				Name: "bob",
				Text: "hi",
			},
		},
	},
	{
		name:  "chat",
		input: "<bob> text|",
		exp: []Message{
			&ChatMessage{
				Name: "bob",
				Text: "text",
			},
		},
	},
	{
		name:  "chat no space",
		input: "<bob>text msg|<fred> msg2|",
		exp: []Message{
			&ChatMessage{
				Name: "bob",
				Text: "text msg",
			},
			&ChatMessage{
				Name: "fred",
				Text: "msg2",
			},
		},
	},
	{
		name:  "chat name with separators",
		input: "<b >b >> text|",
		exp: []Message{
			&ChatMessage{
				Name: "b >b >",
				Text: "text",
			},
		},
	},
	{
		name:  "chat no space and break",
		input: "<bob>some text\r\nthis is formatting>>> some more text|",
		exp: []Message{
			&ChatMessage{
				Name: "bob",
				Text: "some text\r\nthis is formatting>>> some more text",
			},
		},
	},
	{
		name:  "empty chat and trailing",
		input: "<bob>|text" + strings.Repeat(" ", maxName) + "> |",
		exp: []Message{
			&ChatMessage{
				Name: "bob",
			},
			&ChatMessage{
				Text: "text" + strings.Repeat(" ", maxName) + "> ",
			},
		},
	},
	{
		name:  "line break and trailing",
		input: "<bob>\n" + strings.Repeat(" ", maxName) + "> |",
		exp: []Message{
			&ChatMessage{
				Name: "bob",
				Text: strings.Repeat(" ", maxName) + "> ",
			},
		},
	},
	{
		name:  "empty name",
		input: "<> text|",
		exp: []Message{
			&ChatMessage{
				Text: "text",
			},
		},
	},
}

func TestReader(t *testing.T) {
	for _, c := range casesReader {
		t.Run(c.name, func(t *testing.T) {
			r := NewReader(strings.NewReader(c.input))
			var (
				got  []Message
				gerr error
			)
			for {
				m, err := r.ReadMsg()
				if err == io.EOF {
					break
				} else if err != nil {
					gerr = err
					break
				}
				got = append(got, m)
			}
			if c.err == nil {
				require.NoError(t, gerr)
			} else {
				require.Equal(t, c.err, gerr)
			}
			require.Equal(t, c.exp, got, "\n%q\nvs\n%q", c.exp, got)
		})
	}
}

func BenchmarkReader(b *testing.B) {
	const seed = 12345
	rand := rand.New(rand.NewSource(seed))
	const size = 10 * (1 << 20) // 10 MB
	buf := bytes.NewBuffer(nil)
	b.StopTimer()

	cmds := 0
	for buf.Len() < size {
		cmds++
		buf.WriteByte('$')
		buf.WriteString("cmd")
		buf.WriteString(strconv.Itoa(rand.Int()))
		for i := 0; i < rand.Intn(10); i++ {
			buf.WriteByte(' ')
			buf.WriteString(strconv.Itoa(rand.Int()))
		}
		buf.WriteByte(lineDelim)
	}
	data := buf.Bytes()

	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// hide everything except io.Reader
		var r io.Reader = struct {
			io.Reader
		}{bytes.NewReader(data)}

		lr := NewReader(r)

		n := 0
		for {
			_, err := lr.ReadLine()
			if err == io.EOF {
				break
			} else if err != nil {
				b.Fatal(err)
			}
			n++
		}
		if n != cmds {
			b.Fatal("wrong number of commands:", n)
		}
	}
}
