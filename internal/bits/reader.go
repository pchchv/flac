// Package bits provides bit access operations and binary decoding algorithms.
package bits

import (
	"fmt"
	"io"
)

// Reader handles bit reading operations.
// It buffers bits up to the next byte boundary.
type Reader struct {
	r   io.Reader // underlying reader
	buf [8]uint8  // temporary read buffer
	x   uint8     // between 0 and 7 buffered bits since previous read operations
	n   uint      // number of buffered bits in x
}

// NewReader returns a new Reader that reads bits from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Read reads and returns the next n bits, at most 64.
// It buffers bits up to the next byte boundary.
func (br *Reader) Read(n uint) (x uint64, err error) {
	if n == 0 {
		return 0, nil
	}

	if n > 64 {
		return 0, fmt.Errorf("bit.Reader.Read: invalid number of bits; n (%d) exceeds 64", n)
	}

	// Read buffered bits.
	if br.n > 0 {
		switch {
		case br.n == n:
			br.n = 0
			return uint64(br.x), nil
		case br.n > n:
			br.n -= n
			mask := ^uint8(0) << br.n
			x = uint64(br.x&mask) >> br.n
			br.x &^= mask
			return x, nil
		}

		n -= br.n
		x = uint64(br.x)
		br.n = 0
	}

	// Fill the temporary buffer.
	bytes := n / 8
	bits := n % 8
	if bits > 0 {
		bytes++
	}

	if _, err = io.ReadFull(br.r, br.buf[:bytes]); err != nil {
		return 0, err
	}

	// Read bits from the temporary buffer.
	for _, b := range br.buf[:bytes-1] {
		x <<= 8
		x |= uint64(b)
	}

	b := br.buf[bytes-1]
	if bits > 0 {
		x <<= bits
		br.n = 8 - bits
		mask := ^uint8(0) << br.n
		x |= uint64(b&mask) >> br.n
		br.x = b & ^mask
	} else {
		x <<= 8
		x |= uint64(b)
	}

	return x, nil
}
