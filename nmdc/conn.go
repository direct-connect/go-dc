package nmdc

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	SchemeNMDC  = "dchub" // URL scheme for NMDC protocol
	SchemeNMDCS = "nmdcs" // URL scheme for NMDC-over-TLS protocol
	DefaultPort = 411     // default port for client-hub connections
)

// ParseAddr parses an NMDC address an a URL. It will assume a dchub:// scheme if none is set.
func ParseAddr(addr string) (*url.URL, error) {
	if !strings.Contains(addr, "://") {
		addr = SchemeNMDC + "://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != SchemeNMDC && u.Scheme != SchemeNMDCS {
		return u, fmt.Errorf("unsupported protocol: %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}

// NormalizeAddr parses and normalizes the address to scheme://host[:port] format.
func NormalizeAddr(addr string) (string, error) {
	u, err := ParseAddr(addr)
	if err != nil {
		return "", err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		var err2 error
		host, _, err2 = net.SplitHostPort(u.Host + ":" + strconv.Itoa(DefaultPort))
		if err2 != nil {
			return "", err
		}
		err = nil
	}
	if host == "" {
		return "", fmt.Errorf("no hostname in address: %q", addr)
	}
	return u.String(), nil
}
