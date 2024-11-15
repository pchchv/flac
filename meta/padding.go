package meta

import (
	"errors"
	"io"
)

var ErrInvalidPadding = errors.New("invalid padding")

// zeros implements an io.Reader,
// with a Read method which returns
// an error if any byte read isn't zero.
type zeros struct {
	r io.Reader
}

// Read returns an error if any byte read isn't zero.
func (zr zeros) Read(p []byte) (int, error) {
	n, err := zr.r.Read(p)
	for i := 0; i < n; i++ {
		if p[i] != 0 {
			return n, ErrInvalidPadding
		}
	}
	return n, err
}
