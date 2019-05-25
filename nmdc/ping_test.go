package nmdc

import (
	"testing"

	"github.com/direct-connect/go-dc/types"
)

var pingCases = []casesMessageEntry{
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
			Soft: types.Software{
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
			Soft: types.Software{
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
			Soft: types.Software{
				Name:    "YnHub",
				Version: "1.0364",
			},
			Owner: "owner",
			State: "desc 2",
		},
	},
}

func TestPingUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, pingCases)
}

func TestPingMarshal(t *testing.T) {
	doMessageTestMarshal(t, pingCases)
}

func BenchmarkPingUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, pingCases)
}

func BenchmarkPingMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, pingCases)
}
