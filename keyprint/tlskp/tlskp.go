package tlskp

import (
	"crypto/tls"

	"github.com/direct-connect/go-dc/keyprint"
)

// ErrInvalidKeyPrint is returned when an expected keyprint doesn't match any of the peer certificates.
type ErrInvalidKeyPrint struct {
	Expected string
	Actual   []string
}

func (e *ErrInvalidKeyPrint) Error() string {
	return "invalid keyprint"
}

// GetKeyPrints returns keyprints of all certificates provided by a peer on a given TLS connection.
func GetKeyPrints(c *tls.Conn) []string {
	st := c.ConnectionState()
	kps := make([]string, 0, len(st.PeerCertificates))
	for _, cr := range st.PeerCertificates {
		v := keyprint.FromBytes(cr.Raw)
		kps = append(kps, v)
	}
	return kps
}

// VerifyKeyPrint checks if a given keyprint matches any certificates provided by the peer on a given TLS connection.
// It also returns all keyprints for peer certificates.
func VerifyKeyPrint(c *tls.Conn, kp string) ([]string, error) {
	kps := GetKeyPrints(c)
	for _, k := range kps {
		if k == kp {
			return kps, nil
		}
	}
	return kps, &ErrInvalidKeyPrint{Expected: kp, Actual: kps}
}
