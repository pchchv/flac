package bufseekio

import "io"

// ReadSeeker implements buffering for an io.ReadSeeker object.
// ReadSeeker is based on bufio.Reader with
// Seek functionality added and unneeded functionality removed.
type ReadSeeker struct {
	buf []byte
	pos int64         // absolute start position of buf
	rd  io.ReadSeeker // read-seeker provided by the client
	r   int           // buf read positions within buf
	w   int           // buf write positions within buf
	err error
}

// buffered returns the number of bytes that can
// be read from the current buffer.
func (b *ReadSeeker) buffered() int {
	return b.w - b.r
}

func (b *ReadSeeker) readErr() error {
	err := b.err
	b.err = nil
	return err
}
