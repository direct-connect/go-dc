package dc

import (
	"net/url"
	"strings"

	"github.com/direct-connect/go-dc/adc"
	"github.com/direct-connect/go-dc/nmdc"
)

// ParseAddr parses an DC address as a URL.
func ParseAddr(addr string) (*url.URL, error) {
	if strings.HasPrefix(addr, adc.SchemaADC) {
		return adc.ParseAddr(addr)
	}
	return nmdc.ParseAddr(addr)
}

// NormalizeAddr parses and normalizes the address to scheme://host[:port] format.
func NormalizeAddr(addr string) (string, error) {
	if strings.HasPrefix(addr, adc.SchemaADC) {
		return adc.NormalizeAddr(addr)
	}
	return nmdc.NormalizeAddr(addr)
}
