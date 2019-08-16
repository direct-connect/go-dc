package adc

func init() {
	RegisterMessage(RevConnectRequest{})
	RegisterMessage(ConnectRequest{})
}

type RevConnectRequest struct {
	Proto string `adc:"#"`
	Token string `adc:"#"`
}

func (RevConnectRequest) Cmd() MsgType {
	return MsgType{'R', 'C', 'M'}
}

type ConnectRequest struct {
	Proto string `adc:"#"`
	Port  int    `adc:"#"`
	Token string `adc:"#"`
}

func (ConnectRequest) Cmd() MsgType {
	return MsgType{'C', 'T', 'M'}
}
