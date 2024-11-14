// Package bits provides bit access operations and binary decoding algorithms.
package bits

import "io"

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
