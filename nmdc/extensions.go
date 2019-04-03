package nmdc

import (
	"bytes"
	"sort"
)

// http://nmdc.sourceforge.net/NMDC.html#_extensions_commands
// http://nmdc.sourceforge.net/NMDC.html#_extensions_features
// https://www.te-home.net/?do=work&id=verlihub&page=nmdc

// extLockPref is a prefix for a Lock to indicate that Supports handshake can be used.
const extLockPref = "EXTENDEDPROTOCOL"

func init() {
	RegisterMessage(&Supports{})
}

// Supports command lists extensions supported by the peer.
//
// http://nmdc.sourceforge.net/NMDC.html#_supports
type Supports struct {
	Ext []string
}

func (*Supports) Type() string {
	return "Supports"
}

// Intersect two sets of extensions and return a new set as a result.
func (m *Supports) Intersect(m2 *Supports) Supports {
	s := make(map[string]struct{}, len(m.Ext))
	for _, e := range m.Ext {
		s[e] = struct{}{}
	}
	var r Supports
	for _, e := range m2.Ext {
		if _, ok := s[e]; ok {
			r.Ext = append(r.Ext, e)
		}
	}
	return r
}

func (m *Supports) MarshalNMDC(_ *TextEncoder, buf *bytes.Buffer) error {
	n := len(m.Ext) - 1
	for _, ext := range m.Ext {
		n += len(ext)
	}
	buf.Grow(n)
	for i, ext := range m.Ext {
		if i != 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(ext)
	}
	return nil
}

func (m *Supports) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	data = bytes.Trim(data, " ")
	sub := bytes.Split(data, []byte(" "))
	m.Ext = make([]string, 0, len(sub))
	for _, b := range sub {
		if len(b) == 0 {
			continue
		}
		m.Ext = append(m.Ext, string(b))
	}
	return nil
}

// Extensions is an unordered set of protocol extensions.
type Extensions map[string]struct{}

func (f Extensions) Has(name string) bool {
	_, ok := f[name]
	return ok
}

func (f Extensions) Set(name string) {
	f[name] = struct{}{}
}

func (f Extensions) Clone() Extensions {
	f2 := make(Extensions, len(f))
	for name := range f {
		f2[name] = struct{}{}
	}
	return f2
}

func (f Extensions) Intersect(f2 Extensions) Extensions {
	m := make(Extensions)
	for name := range f2 {
		if _, ok := f[name]; ok {
			m[name] = struct{}{}
		}
	}
	return m
}

func (f Extensions) IntersectList(f2 []string) Extensions {
	m := make(Extensions)
	for _, name := range f2 {
		if _, ok := f[name]; ok {
			m[name] = struct{}{}
		}
	}
	return m
}

func (f Extensions) List() []string {
	arr := make([]string, 0, len(f))
	for s := range f {
		arr = append(arr, s)
	}
	sort.Strings(arr)
	return arr
}

// list of known extensions
const (
	ExtNoHello          = "NoHello"
	ExtNoGetINFO        = "NoGetINFO"
	ExtUserCommand      = "UserCommand"
	ExtUserIP2          = "UserIP2"
	ExtTTHSearch        = "TTHSearch"
	ExtZPipe0           = "ZPipe0"
	ExtTLS              = "TLS"
	ExtADCGet           = "ADCGet"
	ExtBotINFO          = "BotINFO"
	ExtHubINFO          = "HubINFO"
	ExtHubTopic         = "HubTopic"
	ExtBotList          = "BotList"
	ExtIN               = "IN"
	ExtMCTo             = "MCTo"
	ExtNickChange       = "NickChange"
	ExtClientNick       = "ClientNick"
	ExtFeaturedNetworks = "FeaturedNetworks"
	ExtGetZBlock        = "GetZBlock"
	ExtClientID         = "ClientID"
	ExtXmlBZList        = "XmlBZList"
	ExtMinislots        = "Minislots"
	ExtTTHL             = "TTHL"
	ExtTTHF             = "TTHF"
	ExtTTHS             = "TTHS"
	ExtZLIG             = "ZLIG"
	ExtACTM             = "ACTM"
	ExtBZList           = "BZList"
	ExtSaltPass         = "SaltPass"
	ExtDHT0             = "DHT0"
	ExtFailOver         = "FailOver"
	ExtOpPlus           = "OpPlus"
	ExtQuickList        = "QuickList"
	ExtBanMsg           = "BanMsg"
	ExtNickRule         = "NickRule"
	ExtSearchRule       = "SearchRule"
	ExtExtJSON2         = "ExtJSON2"
)

// proposals
const (
	ExtLocale = "Locale"
)
