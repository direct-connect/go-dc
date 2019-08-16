package adc

func init() {
	RegisterMessage(ChatMessage{})
}

type ChatMessage struct {
	Text string `adc:"#"`
	PM   *SID   `adc:"PM"`
	Me   bool   `adc:"ME"`
	TS   int64  `adc:"TS"`
}

func (ChatMessage) Cmd() MsgType {
	return MsgType{'M', 'S', 'G'}
}
