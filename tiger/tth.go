package tiger

import "io"

const tthBlock = 1024

// Leaves are a sequence of concatenated hashes that can be used to validate
// the single parts of a certain file.
type Leaves []Hash

// TreeLeaves computes the TTH leaves of a reader.
func TreeLeaves(r io.Reader) (lvl Leaves, err error) {
	buf := make([]byte, tthBlock+1)
	var n int

	for err != io.EOF {
		n, err = r.Read(buf[1:])
		if err != nil && err != io.EOF {
			return
		}
		if n == 0 && len(lvl) > 0 {
			break
		}
		buf[0] = 0x00
		lvl = append(lvl, HashBytes(buf[:n+1]))
	}
	err = nil
	return
}

// TreeHash calculates a Tiger Tree Hash of a reader.
func TreeHash(r io.Reader) (root Hash, err error) {
	var (
		n   int
		lvl []Hash
	)
	buf := make([]byte, tthBlock+1)
	for err != io.EOF {
		n, err = r.Read(buf[1:])
		if err != nil && err != io.EOF {
			return
		}
		if n == 0 && len(lvl) > 0 {
			break
		}
		buf[0] = 0x00
		lvl = append(lvl, HashBytes(buf[:n+1]))
	}
	err = nil
	buf = make([]byte, 2*Size+1)
	for len(lvl) > 1 {
		for i := 0; i < len(lvl); i += 2 {
			if i+1 >= len(lvl) {
				lvl[i/2] = lvl[i]
			} else {
				buf[0] = 0x01
				copy(buf[1:], lvl[i][:])
				copy(buf[1+Size:], lvl[i+1][:])
				lvl[i/2] = HashBytes(buf)
			}
		}
		n := len(lvl) / 2
		if len(lvl)%2 != 0 {
			n++
		}
		lvl = lvl[:n]
	}
	root = lvl[0]
	return
}
