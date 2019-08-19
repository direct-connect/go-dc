package adc

import (
	"bytes"
	"testing"

	"github.com/direct-connect/go-dc/adc/types"
	"github.com/stretchr/testify/require"
)

const delim = "\n"

var casesPackets = []struct {
	name   string
	data   string
	packet Packet
}{
	{
		"BINF empty",
		`BINF AAAB`,
		&BroadcastPacket{
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
			},
			ID: types.SIDFromString("AAAB"),
		},
	},
	{
		"BINF",
		`BINF AAAB IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&BroadcastPacket{
			ID: types.SIDFromString("AAAB"),
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"CINF",
		`CINF IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&ClientPacket{
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"IINF empty",
		`IINF`,
		&InfoPacket{
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
			},
		},
	},
	{
		"IINF",
		`IINF IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&InfoPacket{
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"HINF",
		`HINF IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&HubPacket{
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"DCTM empty",
		`DCTM AAAA BBBB`,
		&DirectPacket{
			ID: types.SIDFromString("AAAA"),
			To: types.SIDFromString("BBBB"),
			Msg: &RawMessage{
				Type: MsgType{'C', 'T', 'M'},
			},
		},
	},
	{
		"DCTM",
		`DCTM AAAA BBBB IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&DirectPacket{
			ID: types.SIDFromString("AAAA"),
			To: types.SIDFromString("BBBB"),
			Msg: &RawMessage{
				Type: MsgType{'C', 'T', 'M'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"EMSG",
		`EMSG AAAA BBBB IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&EchoPacket{
			ID: types.SIDFromString("AAAA"),
			To: types.SIDFromString("BBBB"),
			Msg: &RawMessage{
				Type: MsgType{'M', 'S', 'G'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"UINF empty",
		`UINF KAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI`,
		&UDPPacket{
			ID: types.MustParseCID(`KAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI`),
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
			},
		},
	},
	{
		"UINF",
		`UINF KAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&UDPPacket{
			ID: types.MustParseCID(`KAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI`),
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"FINF empty",
		`FINF AAAB`,
		&FeaturePacket{
			ID: types.SIDFromString("AAAB"),
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
			},
		},
	},
	{
		"FINF",
		`FINF AAAB IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&FeaturePacket{
			ID: types.SIDFromString("AAAB"),
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
	{
		"FINF features empty",
		`FINF AAAB +SEGA -NAT0`,
		&FeaturePacket{
			ID: types.SIDFromString("AAAB"),
			Sel: []FeatureSel{
				{Feature{'S', 'E', 'G', 'A'}, true},
				{Feature{'N', 'A', 'T', '0'}, false},
			},
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
			},
		},
	},
	{
		"FINF features",
		`FINF AAAB +SEGA -NAT0 IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`,
		&FeaturePacket{
			ID: types.SIDFromString("AAAB"),
			Sel: []FeatureSel{
				{Feature{'S', 'E', 'G', 'A'}, true},
				{Feature{'N', 'A', 'T', '0'}, false},
			},
			Msg: &RawMessage{
				Type: MsgType{'I', 'N', 'F'},
				Data: []byte(`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser`),
			},
		},
	},
}

func TestDecodePacket(t *testing.T) {
	for _, c := range casesPackets {
		t.Run(c.name, func(t *testing.T) {
			cmd, err := DecodePacketRaw([]byte(c.data + delim))
			require.NoError(t, err)
			require.Equal(t, c.packet, cmd)
		})
	}
}

func TestEncodePacket(t *testing.T) {
	for _, c := range casesPackets {
		t.Run(c.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			err := c.packet.MarshalPacketADC(buf)
			require.NoError(t, err)
			require.Equal(t, string(c.data+delim), buf.String())
		})
	}
}
