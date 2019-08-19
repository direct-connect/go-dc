package adc

import (
	"testing"
)

var connectCases = []casesMessageEntry{
	{
		"connect request",
		`ADC/1.0 3000 1298498081`,
		&ConnectRequest{Proto: "ADC/1.0", Port: 3000, Token: "1298498081"},
	},
	{
		"rev connect request",
		`ADC/1.0 12345678`,
		&RevConnectRequest{Proto: "ADC/1.0", Token: "12345678"},
	},
}

func TestConnectUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, connectCases)
}

func TestConnectMarshal(t *testing.T) {
	doMessageTestMarshal(t, connectCases)
}
