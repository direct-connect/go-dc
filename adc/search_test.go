package adc

import (
	"testing"
)

var searchCases = []casesMessageEntry{
	{
		"search res",
		`TOtok FNfilepath SI1234567 SL3`,
		&SearchResult{Path: "filepath", Size: 1234567, Slots: 3, Token: "tok"},
	},
	{
		"search par",
		`TO4171511714 ANsome ANdata GR32`,
		&SearchRequest{And: []string{"some", "data"}, Token: "4171511714", Group: ExtVideo},
	},
	{
		"search par2",
		`TO4171511714 GR32`,
		&SearchRequest{Token: "4171511714", Group: ExtVideo},
	},
	{
		"search par3",
		`TO4171511714 ANsome ANdata`,
		&SearchRequest{And: []string{"some", "data"}, Token: "4171511714"},
	},
}

func TestSearchUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, searchCases)
}

func TestSearchMarshal(t *testing.T) {
	doMessageTestMarshal(t, searchCases)
}
