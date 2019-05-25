package nmdc

import (
	"testing"

	"github.com/direct-connect/go-dc/types"
)

var userCases = []casesMessageEntry{
	{
		typ:  "MyINFO",
		data: `$ALL johndoe RU<ApexDC++ V:0.4.0,M:P,H:27/1/3,S:92,L:512>$ $LAN(T3)K$example@example.com$1234$`,
		msg: &MyINFO{
			Name: "johndoe",
			Desc: "RU",
			Client: types.Software{
				Name:    "ApexDC++",
				Version: "0.4.0",
			},
			Mode:           UserModePassive,
			HubsNormal:     27,
			HubsRegistered: 1,
			HubsOperator:   3,
			Slots:          92,
			Extra:          map[string]string{"L": "512"},
			Conn:           "LAN(T3)",
			Flag:           'K',
			Email:          "example@example.com",
			ShareSize:      1234,
		},
	},
	{
		typ:     "MyINFO",
		name:    "no share & no tag",
		data:    `$ALL verg P verg$ $0.005A$$$`,
		expData: `$ALL verg P verg< V:,M: ,H:0/0/0,S:0>$ $0.005A$$0$`,
		msg: &MyINFO{
			Name: "verg",
			Desc: "P verg",
			Mode: UserModeUnknown,
			Conn: "0.005",
			Flag: 'A',
		},
	},
	{
		typ:     "MyINFO",
		name:    "nil field in tag",
		data:    `$ALL whist RU [29]some desc<GreylynkDC++ v:2.3.5,$ $LAN(T1)A$$65075277005$`,
		expData: `$ALL whist RU [29]some desc<GreylynkDC++ V:2.3.5,M: ,H:0/0/0,S:0>$ $LAN(T1)A$$65075277005$`,
		msg: &MyINFO{
			Name: "whist",
			Desc: "RU [29]some desc",
			Client: types.Software{
				Name:    "GreylynkDC++",
				Version: "2.3.5",
			},
			Mode:      UserModeUnknown,
			Conn:      "LAN(T1)",
			Flag:      'A',
			ShareSize: 65075277005,
		},
	},
	{
		typ:     "MyINFO",
		name:    "no vers & hub space",
		data:    `$ALL vespa9347q1 <StrgDC++,M:A,H:1 /0/0,S:2>$ $0.01.$$37038592310$`,
		expData: `$ALL vespa9347q1 <StrgDC++ V:,M:A,H:1/0/0,S:2>$ $0.01.$$37038592310$`,
		msg: &MyINFO{
			Name: "vespa9347q1",
			Client: types.Software{
				Name: "StrgDC++",
			},
			Mode:           UserModeActive,
			HubsNormal:     1,
			HubsRegistered: 0,
			HubsOperator:   0,
			Slots:          2,
			Conn:           "0.01",
			Flag:           '.',
			ShareSize:      37038592310,
		},
	},
	{
		typ:     "MyINFO",
		name:    "only name",
		data:    `$ALL #GlobalOpChat $ $$$0$`,
		expData: "$ALL #GlobalOpChat < V:,M: ,H:0/0/0,S:0>$ $\x01$$0$",
		msg: &MyINFO{
			Name: "#GlobalOpChat",
			Mode: UserModeUnknown,
		},
	},
	{
		typ:     "MyINFO",
		name:    "invalid tag",
		data:    `$ALL test @ HUB-Bot$ $BOT $mail (3.0.1)$BOT $`,
		expData: `$ALL test @ HUB-Bot< V:,M: ,H:0/0/0,S:0>$ $BOT $mail (3.0.1)$0$`,
		msg: &MyINFO{
			Name:  "test",
			Desc:  "@ HUB-Bot",
			Mode:  UserModeUnknown,
			Flag:  FlagTLSUpload,
			Email: "mail (3.0.1)",
			Conn:  "BOT",
		},
	},
	{
		typ:     "MyINFO",
		name:    "legacy P tag",
		data:    `$ALL -EA-Sports $P$$$0$`,
		expData: "$ALL -EA-Sports < V:,M:P,H:0/0/0,S:0>$ $\x01$$0$",
		msg: &MyINFO{
			Name: "-EA-Sports",
			Mode: UserModePassive,
		},
	},
	{
		typ:     "MyINFO",
		name:    "legacy A tag and single hub",
		data:    `$ALL N8611 <++ V:0.868,M:A,H:34,S:3>$A$0.005.$$27225945203$`,
		expData: `$ALL N8611 <++ V:0.868,M:A,H:34/0/0,S:3>$ $0.005.$$27225945203$`,
		msg: &MyINFO{
			Name: "N8611",
			Client: types.Software{
				Name:    "++",
				Version: "0.868",
			},
			Mode:       UserModeActive,
			HubsNormal: 34,
			Slots:      3,
			Conn:       "0.005",
			Flag:       '.',
			ShareSize:  27225945203,
		},
	},
}

func TestUserUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, userCases)
}

func TestUserMarshal(t *testing.T) {
	doMessageTestMarshal(t, userCases)
}

func BenchmarkUserUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, userCases)
}

func BenchmarkUserMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, userCases)
}
