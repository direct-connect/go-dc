package nmdc

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/direct-connect/go-dc"
	"github.com/direct-connect/go-dc/tiger"
)

var casesMessages = []struct {
	typ     string
	name    string
	data    string
	expData string
	msg     Message
}{
	{
		typ:  "FailOver",
		data: `example.com,example.org:5555,adc://example.net:6666`,
		msg: &FailOver{
			Host: []string{
				"example.com",
				"example.org:5555",
				"adc://example.net:6666",
			},
		},
	},
	{
		typ:  "UserIP",
		data: `john doe 192.168.1.2$$`,
		msg: &UserIP{
			Name: "john doe",
			IP:   "192.168.1.2",
		},
	},
	{
		typ:  "Lock",
		name: "no pk no ref",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
		},
	},
	{
		typ:  "Lock",
		name: "without Pk",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC Ref=dchub://example.org:411`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
			Ref:  "dchub://example.org:411",
		},
	},
	{
		typ:  "Lock",
		name: "without Ref",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC Pk=DCPLUSPLUS0.777`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
			PK:   "DCPLUSPLUS0.777",
		},
	},
	{
		typ:  "Lock",
		name: "with Ref",
		data: `EXTENDEDPROTOCOLABCABCABCABCABCABC Pk=DCPLUSPLUS0.777Ref=dchub://example.org:411`,
		msg: &Lock{
			Lock: "ABCABCABCABCABCABC",
			PK:   "DCPLUSPLUS0.777",
			Ref:  "dchub://example.org:411",
		},
	},
	{
		typ:     "HubINFO",
		name:    "9 fields",
		data:    `OZERKI$dc.ozerki.pro$Main Russian D�++ Hub$5000$0$1$2721$PtokaX$`,
		expData: `OZERKI$dc.ozerki.pro$Main Russian D�++ Hub$5000$0$1$2721$PtokaX$$$`,
		msg: &HubINFO{
			Name: "OZERKI",
			Host: "dc.ozerki.pro",
			Desc: "Main Russian D�++ Hub",
			I1:   5000,
			I2:   0,
			I3:   1,
			I4:   2721,
			Soft: dc.Software{
				Name: "PtokaX",
			},
		},
	},
	{
		typ:     "HubINFO",
		name:    "all fields",
		data:    `Angels vs Demons$dc.milenahub.ru$Cogitationis poenam nemo patitur.$20480$0$0$0$Verlihub 1.1.0.12$=FAUST= & KCAHDEP$Public HUB$CP1251`,
		expData: `Angels vs Demons$dc.milenahub.ru$Cogitationis poenam nemo patitur.$20480$0$0$0$Verlihub 1.1.0.12$=FAUST= &amp; KCAHDEP$Public HUB$CP1251`,
		msg: &HubINFO{
			Name: "Angels vs Demons",
			Host: "dc.milenahub.ru",
			Desc: "Cogitationis poenam nemo patitur.",
			I1:   20480,
			I2:   0,
			I3:   0,
			I4:   0,
			Soft: dc.Software{
				Name:    "Verlihub",
				Version: "1.1.0.12",
			},
			Owner:    "=FAUST= & KCAHDEP",
			State:    "Public HUB",
			Encoding: "CP1251",
		},
	},
	{
		typ:     "HubINFO",
		name:    "12 fields",
		data:    `hub name$dc.example.com:8000$hub desc$3000$32212254720$3$40$YnHub 1.0364$owner$desc 2$admin@example.com$`,
		expData: `hub name$dc.example.com:8000$hub desc$3000$32212254720$3$40$YnHub 1.0364$owner$desc 2$`,
		msg: &HubINFO{
			Name: "hub name",
			Host: "dc.example.com:8000",
			Desc: "hub desc",
			I1:   3000,
			I2:   32212254720,
			I3:   3,
			I4:   40,
			Soft: dc.Software{
				Name:    "YnHub",
				Version: "1.0364",
			},
			Owner: "owner",
			State: "desc 2",
		},
	},
	{
		typ:  "MyINFO",
		data: `$ALL johndoe RU<ApexDC++ V:0.4.0,M:P,H:27/1/3,S:92,L:512>$ $LAN(T3)K$example@example.com$1234$`,
		msg: &MyINFO{
			Name: "johndoe",
			Desc: "RU",
			Client: dc.Software{
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
			Client: dc.Software{
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
			Client: dc.Software{
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
			Client: dc.Software{
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
		typ:  "Error",
		data: `message`,
		msg: &Error{
			Err: errors.New("message"),
		},
	},
	{
		typ:  "Search",
		data: `192.168.1.5:412 T?T?500000?1?Gentoo$2005`,
		msg: &Search{
			Address:        "192.168.1.5:412",
			SizeRestricted: true,
			IsMaxSize:      true,
			Size:           500000,
			DataType:       DataTypeAny,
			Pattern:        "Gentoo 2005",
		},
	},
	{
		typ:  "Search",
		name: "TTH",
		data: `Hub:SomeNick F?T?0?9?TTH:TO32WPD6AQE7VA7654HEAM5GKFQGIL7F2BEKFNA`,
		msg: &Search{
			User:           "SomeNick",
			SizeRestricted: false,
			IsMaxSize:      true,
			Size:           0,
			DataType:       DataTypeTTH,
			TTH:            getTHPointer("TO32WPD6AQE7VA7654HEAM5GKFQGIL7F2BEKFNA"),
		},
	},
	{
		typ:     "Search",
		name:    "TTH trailing sep",
		data:    `Hub:SomeNick F?T?0?9?TTH:TO32WPD6AQE7VA7654HEAM5GKFQGIL7F2BEKFNA$`,
		expData: `Hub:SomeNick F?T?0?9?TTH:TO32WPD6AQE7VA7654HEAM5GKFQGIL7F2BEKFNA`,
		msg: &Search{
			User:           "SomeNick",
			SizeRestricted: false,
			IsMaxSize:      true,
			Size:           0,
			DataType:       DataTypeTTH,
			TTH:            getTHPointer("TO32WPD6AQE7VA7654HEAM5GKFQGIL7F2BEKFNA"),
		},
	},
	{
		typ:  "Search",
		name: "TTH trailing sep",
		data: `Hub:SomeNick F?T?0?10?word`,
		msg: &Search{
			User:           "SomeNick",
			SizeRestricted: false,
			IsMaxSize:      true,
			DataType:       DataTypeDiskImage,
			Pattern:        "word",
		},
	},
	{
		typ:     "Search",
		name:    "magnet link",
		data:    `Hub:SomeNick F?T?0?1?magnet:?xt=urn:btih:493C8841D2058D79EA6F7D7103C48D9465B65D41&dn=some$name`,
		expData: `Hub:SomeNick F?T?0?1?magnet:?xt=urn:btih:493C8841D2058D79EA6F7D7103C48D9465B65D41&amp;dn=some$name`,
		msg: &Search{
			User:           "SomeNick",
			SizeRestricted: false,
			IsMaxSize:      true,
			DataType:       DataTypeAny,
			Pattern:        "magnet:?xt=urn:btih:493C8841D2058D79EA6F7D7103C48D9465B65D41&dn=some name",
		},
	},
	{
		typ:  "SA",
		name: "Short TTH search (active)",
		data: `LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ 1.2.3.4:412`,
		msg: &TTHSearchActive{
			TTH:     tiger.MustParseBase32("LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ"),
			Address: "1.2.3.4:412",
		},
	},
	{
		typ:  "SP",
		name: "Short TTH search (passive)",
		data: `LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ user`,
		msg: &TTHSearchPassive{
			TTH:  tiger.MustParseBase32("LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ"),
			User: "user",
		},
	},
	{
		typ:  "SR",
		name: "dir result",
		data: "User6 dir1\\dir 2\\pictures 0/4\x05Testhub (192.168.1.1)",
		msg: &SR{
			From:       "User6",
			Path:       []string{"dir1", "dir 2", "pictures"},
			IsDir:      true,
			TotalSlots: 4,
			HubName:    "Testhub",
			HubAddress: "192.168.1.1",
		},
	},
	{
		typ:  "SR",
		name: "file result",
		data: "User1 dir\\file 1.txt\x05437 3/4\x05Testhub (192.168.1.1:411)\x05User2",
		msg: &SR{
			From:       "User1",
			Path:       []string{"dir", "file 1.txt"},
			Size:       437,
			FreeSlots:  3,
			TotalSlots: 4,
			HubName:    "Testhub",
			HubAddress: "192.168.1.1:411",
			To:         "User2",
		},
	},
	{
		typ:  "SR",
		name: "tth result",
		data: "User1 Linux\\kubuntu-18.04-desktop-amd64.iso\x051868038144 3/3\x05TTH:BNQGWMXKUIAFAU3TV32I5U6SKNYMQBBNH4FELNQ (192.168.1.1:411)\x05User2",
		msg: &SR{
			From:       "User1",
			Path:       []string{"Linux", "kubuntu-18.04-desktop-amd64.iso"},
			Size:       1868038144,
			FreeSlots:  3,
			TotalSlots: 3,
			TTH:        getTHPointer("BNQGWMXKUIAFAU3TV32I5U6SKNYMQBBNH4FELNQ"),
			HubAddress: "192.168.1.1:411",
			To:         "User2",
		},
	},
	{
		typ:  "SR",
		name: "space in path",
		data: "User1 dir\\some file.dat\x05152374784 1/3\x05TTH:HRFQOVMYIGSSGXN4FDTOGWO4USC24BBVQLOKIQI (1.2.3.4:411)\x05User2",
		msg: &SR{
			From:       "User1",
			Path:       []string{"dir", "some file.dat"},
			Size:       152374784,
			FreeSlots:  1,
			TotalSlots: 3,
			TTH:        getTHPointer("HRFQOVMYIGSSGXN4FDTOGWO4USC24BBVQLOKIQI"),
			HubAddress: "1.2.3.4:411",
			To:         "User2",
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

func getTHPointer(s string) *tiger.Hash {
	pointer := tiger.MustParseBase32(s)
	return &pointer
}

func TestUnmarshal(t *testing.T) {
	for _, c := range casesMessages {
		name := c.typ
		if c.name != "" {
			name += " " + c.name
		}
		t.Run(name, func(t *testing.T) {
			m := NewMessage(c.typ)
			err := m.UnmarshalNMDC(nil, []byte(c.data))
			require.NoError(t, err)
			require.Equal(t, c.msg, m)
		})
	}
}

func TestMarshal(t *testing.T) {
	for _, c := range casesMessages {
		name := c.typ
		if c.name != "" {
			name += " " + c.name
		}
		t.Run(name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			err := c.msg.MarshalNMDC(nil, buf)
			exp := c.expData
			if exp == "" {
				exp = c.data
			}
			require.NoError(t, err)
			require.Equal(t, exp, buf.String())
		})
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	for _, c := range casesMessages {
		name := c.typ
		if c.name != "" {
			name += " " + c.name
		}
		b.Run(name, func(b *testing.B) {
			data := []byte(c.data)
			for i := 0; i < b.N; i++ {
				m := NewMessage(c.typ)
				err := m.UnmarshalNMDC(nil, data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMarshal(b *testing.B) {
	for _, c := range casesMessages {
		name := c.typ
		if c.name != "" {
			name += " " + c.name
		}
		b.Run(name, func(b *testing.B) {
			m := NewMessage(c.typ)
			err := m.UnmarshalNMDC(nil, []byte(c.data))
			buf := bytes.NewBuffer(nil)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				buf.Reset()
				err = m.MarshalNMDC(nil, buf)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
