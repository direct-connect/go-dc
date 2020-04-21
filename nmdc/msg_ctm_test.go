package nmdc

import (
	"testing"
)

var connectCases = []casesMessageEntry{
	{
		typ:  "ConnectToMe",
		data: `john 192.168.1.2:412`,
		msg: &ConnectToMe{
			Targ:    "john",
			Address: "192.168.1.2:412",
			Kind:    CTMActive,
			Secure:  false,
		},
	},
	{
		typ:  "ConnectToMe",
		data: `john 192.168.1.2:412S`,
		msg: &ConnectToMe{
			Targ:    "john",
			Address: "192.168.1.2:412",
			Kind:    CTMActive,
			Secure:  true,
		},
	},
	{
		typ:  "ConnectToMe",
		data: `john 192.168.1.2:412N peter`,
		msg: &ConnectToMe{
			Targ:    "john",
			Src:     "peter",
			Address: "192.168.1.2:412",
			Kind:    CTMPassiveReq,
			Secure:  false,
		},
	},
	{
		typ:  "ConnectToMe",
		data: `john 192.168.1.2:412NS peter`,
		msg: &ConnectToMe{
			Targ:    "john",
			Src:     "peter",
			Address: "192.168.1.2:412",
			Kind:    CTMPassiveReq,
			Secure:  true,
		},
	},
	{
		typ:  "ConnectToMe",
		data: `john 192.168.1.2:412R`,
		msg: &ConnectToMe{
			Targ:    "john",
			Address: "192.168.1.2:412",
			Kind:    CTMPassiveResp,
			Secure:  false,
		},
	},
	{
		typ:  "ConnectToMe",
		data: `john 192.168.1.2:412RS`,
		msg: &ConnectToMe{
			Targ:    "john",
			Address: "192.168.1.2:412",
			Kind:    CTMPassiveResp,
			Secure:  true,
		},
	},
}

func TestConnectUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, connectCases)
}

func TestConnectMarshal(t *testing.T) {
	doMessageTestMarshal(t, connectCases)
}

func BenchmarkConnectUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, connectCases)
}

func BenchmarkConnectMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, connectCases)
}
