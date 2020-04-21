package nmdc

import (
	"testing"
)

var chatCases = []casesMessageEntry{
	{
		typ:  "To:",
		data: `john From: room $<peter> dogs are more cute`,
		msg: &PrivateMessage{
			To:   "john",
			From: "room",
			Name: "peter",
			Text: "dogs are more cute",
		},
	},
	{
		typ:  "To:",
		data: `user 1 From: room 1 $<user 2> private message`,
		msg: &PrivateMessage{
			To:   "user 1",
			From: "room 1",
			Name: "user 2",
			Text: "private message",
		},
	},
	{
		typ:  "MCTo",
		data: `target $sender some message`,
		msg: &MCTo{
			To:   "target",
			From: "sender",
			Text: "some message",
		},
	},
	{
		typ:  "UserCommand",
		name: "raw",
		data: `1 3 # Ledokol Menu\.:: Ranks\All time user location statistics $<%[mynick]> +cchist`,
		msg: &UserCommand{
			Typ:     TypeRaw,
			Context: ContextHub | ContextUser,
			Path:    []string{"# Ledokol Menu", ".:: Ranks", "All time user location statistics"},
			Command: "<%[mynick]> +cchist",
		},
	},
	{
		typ:     "UserCommand",
		name:    "raw",
		data:    `1 3 a\b\c$<%[mynick]> +cchist`,
		expData: `1 3 a\b\c $<%[mynick]> +cchist`,
		msg: &UserCommand{
			Typ:     TypeRaw,
			Context: ContextHub | ContextUser,
			Path:    []string{"a", "b", "c"},
			Command: "<%[mynick]> +cchist",
		},
	},
	{
		typ:  "UserCommand",
		name: "erase",
		data: `255 1`,
		msg: &UserCommand{
			Typ:     TypeErase,
			Context: ContextHub,
		},
	},
	{
		typ:     "UserCommand",
		name:    "erase with space",
		data:    `255 1 `,
		expData: `255 1`,
		msg: &UserCommand{
			Typ:     TypeErase,
			Context: ContextHub,
		},
	},
	{
		typ:     "UserCommand",
		name:    "escaped",
		data:    `0 3&#124;`,
		expData: `0 3`,
		msg: &UserCommand{
			Typ:     TypeSeparator,
			Context: ContextHub | ContextUser,
		},
	},
}

func TestChatUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, chatCases)
}

func TestChatMarshal(t *testing.T) {
	doMessageTestMarshal(t, chatCases)
}

func BenchmarkChatUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, chatCases)
}

func BenchmarkChatMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, chatCases)
}
