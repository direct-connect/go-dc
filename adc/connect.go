package adc

func init() {
	RegisterMessage(RevConnectRequest{})
	RegisterMessage(ConnectRequest{})
}

var _ Message = RevConnectRequest{}

type RevConnectRequest struct {
	Proto string `adc:"#"`
	Token string `adc:"#"`
}

func (RevConnectRequest) Cmd() MsgType {
	return MsgType{'R', 'C', 'M'}
}

var _ Message = ConnectRequest{}

type ConnectRequest struct {
	Proto string `adc:"#"`
	Port  int    `adc:"#"`
	Token string `adc:"#"`
}

func (ConnectRequest) Cmd() MsgType {
	return MsgType{'C', 'T', 'M'}
}
