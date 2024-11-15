package meta

import "io"

// zeros implements an io.Reader,
// with a Read method which returns
// an error if any byte read isn't zero.
type zeros struct {
	r io.Reader
}
