package adc

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

const (
	ProtoADC  = `ADC/1.0`   // ADC protocol name in CTM requests
	ProtoADCS = `ADCS/0.10` // ADCS protocol name in CTM requests

	SchemaADC  = "adc"  // URL scheme for ADC protocol
	SchemaADCS = "adcs" // URL scheme for ADC-over-TLS protocol
)

// ParseAddr parses an ADC address as a URL.
func ParseAddr(addr string) (*url.URL, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != SchemaADC && u.Scheme != SchemaADCS {
		return u, fmt.Errorf("unsupported protocol: %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}

// NormalizeAddr parses and normalizes the address to scheme://host:port format.
func NormalizeAddr(addr string) (string, error) {
	u, err := ParseAddr(addr)
	if err != nil {
		return "", err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", fmt.Errorf("failed to parse host-port pair: %v", err)
	} else if host == "" {
		return "", fmt.Errorf("no hostname in address: %q", addr)
	}
	return u.String(), nil
}
