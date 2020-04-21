package nmdc

import (
	"testing"

	"github.com/direct-connect/go-dc/tiger"
)

var searchCases = []casesMessageEntry{
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
}

func TestSearchUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, searchCases)
}

func TestSearchMarshal(t *testing.T) {
	doMessageTestMarshal(t, searchCases)
}

func BenchmarkSearchUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, searchCases)
}

func BenchmarkSearchMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, searchCases)
}
