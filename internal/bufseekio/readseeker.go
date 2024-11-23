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
