package tiger

import (
	"bytes"
	"reflect"
	"testing"
)

var tthCases = []struct {
	data   []byte
	leaves Leaves
	hash   string
}{
	{
		[]byte{},
		Leaves([]Hash{MustParseBase32("LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ")}),
		`LWPNACQDBZRYXW3VHJVCJ64QBZNGHOHHHZWCLNQ`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1),
		Leaves([]Hash{MustParseBase32("CZQUWH3IYXBF5L3BGYUGZHASSMXU647IP2IKE4Y")}),
		`CZQUWH3IYXBF5L3BGYUGZHASSMXU647IP2IKE4Y`,
	},
	{
		bytes.Repeat([]byte{'a'}, 5),
		Leaves([]Hash{MustParseBase32("ELXBTR33AWAAEKEVWRXEQ3446IL7KGCTXMWA4AA")}),
		`ELXBTR33AWAAEKEVWRXEQ3446IL7KGCTXMWA4AA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 24),
		Leaves([]Hash{MustParseBase32("K56WCQPI62DYXXDY4AZ7LRUFDQOTIZRAPEKRTRI")}),
		`K56WCQPI62DYXXDY4AZ7LRUFDQOTIZRAPEKRTRI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 25),
		Leaves([]Hash{MustParseBase32("BNCXPH7SJ5Z4HTKEYMJXFL7QJUXLZFZM4JDRQYY")}),
		`BNCXPH7SJ5Z4HTKEYMJXFL7QJUXLZFZM4JDRQYY`,
	},
	{
		bytes.Repeat([]byte{'a'}, 64),
		Leaves([]Hash{MustParseBase32("LKOML52BOHG43N2P5MNZ3BDIAKNYO3C22WQMJGI")}),
		`LKOML52BOHG43N2P5MNZ3BDIAKNYO3C22WQMJGI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 100),
		Leaves([]Hash{MustParseBase32("MI3GUSIV63KCZS4IL3PEZD6AQADVO6CMKPITPTA")}),
		`MI3GUSIV63KCZS4IL3PEZD6AQADVO6CMKPITPTA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 127),
		Leaves([]Hash{MustParseBase32("YKSDLGFJM7HNVU3ESUVCOT4JGPB2NWL3WIMPLZA")}),
		`YKSDLGFJM7HNVU3ESUVCOT4JGPB2NWL3WIMPLZA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 128),
		Leaves([]Hash{MustParseBase32("3ZTFBW4Y65OGGNXCM776DYN5WJ6SZLWR7WMC4NA")}),
		`3ZTFBW4Y65OGGNXCM776DYN5WJ6SZLWR7WMC4NA`,
	},
	{
		bytes.Repeat([]byte{'a'}, 256),
		Leaves([]Hash{MustParseBase32("ZZK5ZBTLKGLY7SFWEHY5VGYYDQHZG56NIUQ6IXI")}),
		`ZZK5ZBTLKGLY7SFWEHY5VGYYDQHZG56NIUQ6IXI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1022),
		Leaves([]Hash{MustParseBase32("PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I")}),
		`PT2BL57H4JJ5LHXBDA6CJ5KEOO5XEKNIFYINE7I`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1023),
		Leaves([]Hash{MustParseBase32("YBJDV4HQU6LDJZMP36DEUZ7MMNXA6TBLMOX55PI")}),
		`YBJDV4HQU6LDJZMP36DEUZ7MMNXA6TBLMOX55PI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1024),
		Leaves([]Hash{MustParseBase32("BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI")}),
		`BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI`,
	},
	{
		bytes.Repeat([]byte{'a'}, 1025),
		Leaves([]Hash{
			MustParseBase32("BR4BVJBMHDFVCFI4WBPSL63W5TWXWVBSC574BLI"),
			MustParseBase32("CZQUWH3IYXBF5L3BGYUGZHASSMXU647IP2IKE4Y"),
		}),
		`CDYY2OW6F6DTGCH3Q6NMSDLSRV7PNMAL3CED3DA`,
	},
}

func TestTTHLeaves(t *testing.T) {
	for i, c := range tthCases {
		lvl, err := TreeLeaves(bytes.NewReader(c.data))
		if err != nil {
			t.Fatal(err)
		} else if reflect.DeepEqual(lvl, c.leaves) == false {
			t.Errorf("wrong leaves on %d: %v vs %v", i+1, c.leaves, lvl)
		}
	}
}

func TestTTHLeavesToTreeHash(t *testing.T) {
	for i, c := range tthCases {
		lvl, err := TreeLeaves(bytes.NewReader(c.data))
		if err != nil {
			t.Fatal(err)
		}
		h := lvl.TreeHash()
		if h != MustParseBase32(c.hash) {
			t.Errorf("wrong hash on %d: %s vs %s", i+1, c.hash, h)
		}
	}
}

func TestTTH(t *testing.T) {
	for i, c := range tthCases {
		tr, err := TreeHash(bytes.NewReader(c.data))
		if err != nil {
			t.Fatal(err)
		} else if c.hash != tr.String() {
			t.Errorf("wrong hash on %d: %s vs %s", i+1, c.hash, tr)
		}
	}
}
