package adc

import (
	"testing"
)

var chatCases = []casesMessageEntry{
	{
		"msg",
		`some\stext`,
		&ChatMessage{Text: "some text"},
	},
	{
		"pm",
		`some\stext PMAAAB`,
		&ChatMessage{Text: "some text", PM: sidp("AAAB")},
	},
	{
		"me",
		`some\stext ME1`,
		&ChatMessage{Text: "some text", Me: true},
	},
}

func TestChatUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, chatCases)
}

func TestChatMarshal(t *testing.T) {
	doMessageTestMarshal(t, chatCases)
}
