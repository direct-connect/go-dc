package adc

import (
	"testing"

	"github.com/direct-connect/go-dc/adc/types"
)

var userCases = []casesMessageEntry{
	{
		"user",
		`IDHVBNEMDCTKCD4V3N54X4MMOVLJLJL6PSKVHFXHI NIgopher I4172.17.42.1 SS39542721391 SF34 VEGoConn\s0.01 SL3 FS0 HN0 HR0 HO0 SUGCON,ADC0`,
		&UserInfo{
			Id:         types.MustParseCID(`HVBNEMDCTKCD4V3N54X4MMOVLJLJL6PSKVHFXHI`),
			Name:       "gopher",
			Ip4:        "172.17.42.1",
			ShareFiles: 34,
			ShareSize:  39542721391,
			Version:    `GoConn 0.01`,
			Slots:      3,
			Features:   ExtFeatures{{'G', 'C', 'O', 'N'}, {'A', 'D', 'C', '0'}},
		},
	},
	{
		"user id",
		`IDHVBNEMDCTKCD4V3N54X4MMOVLJLJL6PSKVHFXHI PDHVBNEMDCTKCD4V3N54X4MMOVLJLJL6PSKVHFXHI NIgopher I4172.17.42.1 SS39542721391 SF34 VEGoConn\s0.01 SL3 FS0 HN0 HR0 HO0 SUGCON,ADC0`,
		&UserInfo{
			Id:         types.MustParseCID(`HVBNEMDCTKCD4V3N54X4MMOVLJLJL6PSKVHFXHI`),
			Pid:        types.MustParseCIDP(`HVBNEMDCTKCD4V3N54X4MMOVLJLJL6PSKVHFXHI`),
			Name:       "gopher",
			Ip4:        "172.17.42.1",
			ShareFiles: 34,
			ShareSize:  39542721391,
			Version:    `GoConn 0.01`,
			Slots:      3,
			Features:   ExtFeatures{{'G', 'C', 'O', 'N'}, {'A', 'D', 'C', '0'}},
		},
	},
	{
		"user pid",
		`IDKAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI NIuser I4172.17.42.1 U43000 SS25146919163 SF23 VEEiskaltDC++\s2.2.9 US1310720 SL3 FS3 HN11 HR0 HO1 SUSEGA,ADC0,TCP4,UDP4 KPSHA256/C44JWX62IN6JBAVH7NIHEZIQ6WSNQ2LHTOWYWP7ADGAYTCPZVWRQ`,
		&UserInfo{
			Id:           types.MustParseCID(`KAY6BI76T6XFIQXZNRYE4WXJ2Y3YGXJG7UM7XLI`),
			Name:         "user",
			Ip4:          "172.17.42.1",
			ShareFiles:   23,
			ShareSize:    25146919163,
			Version:      `EiskaltDC++ 2.2.9`,
			Udp4:         3000,
			MaxUpload:    "1310720",
			Slots:        3,
			SlotsFree:    3,
			HubsNormal:   11,
			HubsOperator: 1,
			Features:     ExtFeatures{{'S', 'E', 'G', 'A'}, {'A', 'D', 'C', '0'}, {'T', 'C', 'P', '4'}, {'U', 'D', 'P', '4'}},
			KP:           "SHA256/C44JWX62IN6JBAVH7NIHEZIQ6WSNQ2LHTOWYWP7ADGAYTCPZVWRQ",
		},
	},
	{
		"user no name",
		`NI SS34815324082 SF8416 VE SL0 FS1 HN18 HR0 HO2 SUNAT0,ADC0,SEGA`,
		&UserInfo{
			ShareFiles:   8416,
			ShareSize:    34815324082,
			HubsNormal:   18,
			HubsOperator: 2,
			SlotsFree:    1,
			Udp4:         0,
			Features:     ExtFeatures{{'N', 'A', 'T', '0'}, {'A', 'D', 'C', '0'}, {'S', 'E', 'G', 'A'}},
		},
	},
}

func TestUserUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, userCases)
}

func TestUserMarshal(t *testing.T) {
	doMessageTestMarshal(t, userCases)
}
