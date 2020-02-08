package keyprint

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/base32"
	"encoding/pem"
	"errors"
	"io"
	"io/ioutil"
	"net/url"
)

// FromCertificate returns keyprints of all certificates in cert.
func FromCertificate(cert tls.Certificate) []string {
	if len(cert.Certificate) == 0 {
		return nil
	}
	kps := make([]string, 0, len(cert.Certificate))
	for _, c := range cert.Certificate {
		kps = append(kps, FromBytes(c))
	}
	return kps
}

var base32enc = base32.StdEncoding.WithPadding(base32.NoPadding)

// FromBytes calculates a SHA256 keyprint from cert bytes.
// This function cannot be used to parse cert files, use FromFile instead.
func FromBytes(b []byte) string {
	h := sha256.Sum256(b)
	const pref = "SHA256/"
	out := make([]byte, len(pref)+base32enc.EncodedLen(len(h)))
	copy(out, pref)
	base32enc.Encode(out[len(pref):], h[:])
	return string(out)
}

// FromBytes calculates a SHA256 keyprint from a PEM cert file.
func FromFile(r io.Reader) ([]string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var kps []string
	for {
		b, rest := pem.Decode(data)
		if b == nil {
			break
		}
		data = rest
		if b.Type != "CERTIFICATE" {
			continue
		}
		kps = append(kps, FromBytes(b.Bytes))
	}
	if len(kps) == 0 {
		return nil, errors.New("invalid PEM file, or missing a CERTIFICATE block")
	}
	return kps, nil
}

// FromURL extracts a keyprint from URL query parameters. If it's not set, it returns an empty string.
func FromURL(u *url.URL) string {
	return u.Query().Get("kp")
}
