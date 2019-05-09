package adc

const (
	FileListBZIP = "files.xml.bz2"
)

func init() {
	RegisterMessage(GetInfoRequest{})
	RegisterMessage(GetRequest{})
	RegisterMessage(GetResponse{})
}

var _ Message = GetInfoRequest{}

type GetInfoRequest struct {
	Type string `adc:"#"`
	Path string `adc:"#"`
}

func (GetInfoRequest) Cmd() MsgType {
	return MsgType{'G', 'F', 'I'}
}

var _ Message = GetRequest{}

type GetRequest struct {
	Type  string `adc:"#"`
	Path  string `adc:"#"`
	Start int64  `adc:"#"`
	Bytes int64  `adc:"#"`
}

func (GetRequest) Cmd() MsgType {
	return MsgType{'G', 'E', 'T'}
}

var _ Message = GetResponse{}

type GetResponse GetRequest

func (GetResponse) Cmd() MsgType {
	return MsgType{'S', 'N', 'D'}
}
