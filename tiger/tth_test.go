package tiger_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/direct-connect/go-dc/tiger"
)

var tthCases = []struct {
	data   []byte
	leaves tiger.Leaves
	hash   string
}{
	{
		[]byte{},
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ")}),
		`LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("CZQUWH3IYXBF5L3BGYUGZHASSMXU647IP2IKE4Y")}),
		`CZQUWH3IYXBF5L3BGYUGZHASSMXU647IP2IKE4Y`,
	},
	{
		bytes.Repeat([]byte{'a'}, 5),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("ELXBTR33AWAAEKEVWRXEQ3446IL7KGCTXMWA4AA")}),
		`ELXBTR33AWAAEKEVWRXEQ3446IL7KGCTXMWA4AA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 24),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("K56WCQPI62DYXXDY4AZ7LRUFDQOTIZRAPEKRTRI")}),
		`K56WCQPI62DYXXDY4AZ7LRUFDQOTIZRAPEKRTRI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 25),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("BNCXPH7SJ5Z4HTKEYMJXFL7QJUXLZFZM4JDRQYY")}),
		`BNCXPH7SJ5Z4HTKEYMJXFL7QJUXLZFZM4JDRQYY`,
	},
	{
		bytes.Repeat([]byte{'a'}, 64),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("LKOML52BOHG43N2P5MNZ3BDIAKNYO3C22WQMJGI")}),
		`LKOML52BOHG43N2P5MNZ3BDIAKNYO3C22WQMJGI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 100),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("MI3GUSIV63KCZS4IL3PEZD6AQADVO6CMKPITPTA")}),
		`MI3GUSIV63KCZS4IL3PEZD6AQADVO6CMKPITPTA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 127),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("YKSDLGFJM7HNVU3ESUVCOT4JGPB2NWL3WIMPLZA")}),
		`YKSDLGFJM7HNVU3ESUVCOT4JGPB2NWL3WIMPLZA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 128),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("3ZTFBW4Y65OGGNXCM776DYN5WJ6SZLWR7WMC4NA")}),
		`3ZTFBW4Y65OGGNXCM776DYN5WJ6SZLWR7WMC4NA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 256),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("ZZK5ZBTLKGLY7SFWEHY5VGYYDQHZG56NIUQ6IXI")}),
		`ZZK5ZBTLKGLY7SFWEHY5VGYYDQHZG56NIUQ6IXI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1022),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I")}),
		`PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1023),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("YBJDV4HQU6LDJZMP36DEUZ7MMNXA6TBLMOX55PI")}),
		`YBJDV4HQU6LDJZMP36DEUZ7MMNXA6TBLMOX55PI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1024),
		tiger.Leaves([]tiger.Hash{tiger.MustParseBase32("BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI")}),
		`BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1025),
		tiger.Leaves([]tiger.Hash{
			tiger.MustParseBase32("BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI"),
			tiger.MustParseBase32("CZQUWH3IYXBF5L3BGYUGZHASSMXU647IP2IKE4Y"),
		}),
		`CDYY2OW6F6DTGCH3Q6NMSDLSRV7PNMAL3CED3DA`,
	},
}

func TestTTHLeaves(t *testing.T) {
	for i, c := range tthCases {
		lvl, err := tiger.TreeLeaves(bytes.NewReader(c.data))
		if err != nil {
			t.Fatal(err)
		} else if reflect.DeepEqual(lvl, c.leaves) == false {
			t.Errorf("wrong leaves on %d: %v vs %v", i+1, c.leaves, lvl)
		}
	}
}

func TestTTHLeavesToTreeHash(t *testing.T) {
	for i, c := range tthCases {
		lvl, err := tiger.TreeLeaves(bytes.NewReader(c.data))
		if err != nil {
			t.Fatal(err)
		}
		h := lvl.TreeHash()
		if h != tiger.MustParseBase32(c.hash) {
			t.Errorf("wrong hash on %d: %s vs %s", i+1, c.hash, h)
		}
	}
}

func TestTTH(t *testing.T) {
	for i, c := range tthCases {
		tr, err := tiger.TreeHash(bytes.NewReader(c.data))
		if err != nil {
			t.Fatal(err)
		} else if c.hash != tr.String() {
			t.Errorf("wrong hash on %d: %s vs %s", i+1, c.hash, tr)
		}
	}
}
