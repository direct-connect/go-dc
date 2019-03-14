// Package tiger implements Tiger hash function and TTH (Tiger Tree Hash) algorithm.
package tiger

import (
	"encoding"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"hash"

	th "github.com/cxmcc/tiger"
)

// New returns a new hash.Hash that calculates the Tiger/192 hash digest.
func New() hash.Hash {
	return th.New()
}

const (
	BlockSize = 64 // 512 bits
	Size      = 24 // 192 bits
)

// HashBytes calculates the tiger hash of a byte slice.
func HashBytes(b []byte) (out Hash) {
	h := New()
	if _, err := h.Write(b); err != nil {
		panic(err)
	}
	h.Sum(out[:0])
	return
}

var base32Enc = base32.StdEncoding.WithPadding(base32.NoPadding)

// MustParseBase32 parses the tiger hash from base32 encoding and panics on error.
func MustParseBase32(s string) (out Hash) {
	if err := out.FromBase32(s); err != nil {
		panic(err)
	}
	return
}

var (
	_ encoding.TextMarshaler   = (*Hash)(nil)
	_ encoding.TextUnmarshaler = (*Hash)(nil)
)

var zeroTH = Hash{}

// Hash is a tiger hash value.
type Hash [Size]byte

// IsZero check if hash value is zero.
func (h Hash) IsZero() bool { return h == zeroTH }

// Bytes returns byte slice from the hash. Same as h[:].
func (h Hash) Bytes() []byte { return h[:] }

// String returns base32 representation of the hash.
func (h Hash) String() string { return h.Base32() }

// Hex returns hexadecimal representation of the hash.
func (h Hash) Hex() string {
	return hex.EncodeToString(h[:])
}

// Base32 returns base32 representation of the hash.
func (h Hash) Base32() string {
	return base32Enc.EncodeToString(h[:])
}

// FromBase32 parses hash from base32 encoding.
func (h *Hash) FromBase32(s string) error {
	b, err := base32Enc.DecodeString(s)
	if err != nil {
		return err
	}
	if len(b) != Size {
		return fmt.Errorf("wrong base32 encoded size: %d vs %d", len(b), Size)
	}
	copy((*h)[:], b)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (h Hash) MarshalText() ([]byte, error) {
	return []byte(h.Base32()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (h *Hash) UnmarshalText(text []byte) error {
	return h.FromBase32(string(text))
}

func (h *Hash) UnmarshalAdc(s []byte) error {
	return h.FromBase32(string(s))
}
func (h Hash) MarshalAdc() ([]byte, error) {
	return []byte(h.Base32()), nil
}
