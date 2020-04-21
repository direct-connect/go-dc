package nmdc

import (
	"bytes"
	"errors"
	"math"
	"strconv"
)

func isASCII(p []byte) bool {
	for _, b := range p {
		if b == '/' || b == '-' || b == '_' || b == '.' || b == ':' {
			continue
		}
		if b < '0' || b > 'z' {
			return false
		}
		if b >= 'a' && b <= 'z' {
			continue
		}
		if b >= 'A' && b <= 'Z' {
			continue
		}
		if b >= '0' && b <= '9' {
			continue
		}
		return false
	}
	return true
}

func trimSpace(s []byte) []byte {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' {
			s = s[i:]
			break
		}
	}
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != ' ' {
			s = s[:i+1]
			break
		}
	}
	return s
}

func atoiTrim(s []byte) (int, error) {
	sLen := len(s)
	if sLen == 0 {
		return 0, strconv.ErrSyntax
	}
	s = trimSpace(s)
	sLen = len(s)
	if sLen == 0 {
		return 0, strconv.ErrSyntax
	}
	// fast path from strconv.Atoi
	if sLen < 10 {
		// Fast path for small integers that fit int type.
		s0 := s
		if s[0] == '-' || s[0] == '+' {
			s = s[1:]
			if len(s) < 1 {
				return 0, strconv.ErrSyntax
			}
		}
		n := 0
		for _, ch := range s {
			ch -= '0'
			if ch > 9 {
				return 0, strconv.ErrSyntax
			}
			n = n*10 + int(ch)
		}
		if s0[0] == '-' {
			n = -n
		}
		return n, nil
	}
	return strconv.Atoi(string(s))
}

func parseUin64Trim(s []byte) (uint64, error) {
	if len(s) == 0 {
		return 0, strconv.ErrSyntax
	}
	s = trimSpace(s)
	if len(s) == 0 {
		return 0, strconv.ErrSyntax
	}
	cutoff := uint64(math.MaxUint64/10 + 1)

	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, strconv.ErrSyntax
		}
		d := c - '0'
		if n >= cutoff {
			// n*base overflows
			return 0, strconv.ErrSyntax
		}
		n *= 10

		n1 := n + uint64(d)
		if n1 < n || n1 > math.MaxUint64 {
			// n+v overflows
			return 0, strconv.ErrSyntax
		}
		n = n1
	}
	return n, nil
}

func splitN(p []byte, sep byte, n int) ([][]byte, bool) {
	c := bytes.Count(p, []byte{sep})
	if c != n-1 {
		return nil, false
	}
	out := make([][]byte, 0, n)
	for i := 0; i < c; i++ {
		j := bytes.IndexByte(p, sep)
		out = append(out, p[:j])
		p = p[j+1:]
	}
	out = append(out, p)
	return out, true
}

func escapeString(sw *bytes.Buffer, s string) error {
	last := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == 0x00 {
			return errors.New("invalid characters in string")
		} else if escapeCharsString[b] == "" {
			continue
		}
		if last != i {
			sw.WriteString(s[last:i])
		}
		last = i + 1
		sw.WriteString(escapeCharsString[b])
	}
	if last != len(s) {
		sw.WriteString(s[last:])
	}
	return nil
}

func escapeName(sw *bytes.Buffer, s string) error {
	last := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if invalidCharsNameI[b] {
			return errors.New("invalid characters in name")
		} else if escapeCharsName[b] == "" {
			continue
		}
		if last != i {
			sw.WriteString(s[last:i])
		}
		last = i + 1
		sw.WriteString(escapeCharsName[b])
	}
	if last != len(s) {
		sw.WriteString(s[last:])
	}
	return nil
}
