package bufseekio

import (
	"errors"
	"io"
)

const (
	defaultBufSize    = 4096
	minReadBufferSize = 16
)

var errNegativeRead = errors.New("bufseekio: reader returned negative count from Read")

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

// NewReadSeekerSize returns a
// new ReadSeeker whose buffer has at least the specified size.
// If the argument io.ReadSeeker is already a
// ReadSeeker with large enough size,
// it returns the underlying ReadSeeker.
func NewReadSeekerSize(rd io.ReadSeeker, size int) *ReadSeeker {
	// Is it already a Reader?
	if b, ok := rd.(*ReadSeeker); ok && len(b.buf) >= size {
		return b
	}

	if size < minReadBufferSize {
		size = minReadBufferSize
	}

	r := new(ReadSeeker)
	r.reset(make([]byte, size), rd)
	return r
}

// NewReadSeeker returns a new ReadSeeker whose buffer has the default size.
func NewReadSeeker(rd io.ReadSeeker) *ReadSeeker {
	return NewReadSeekerSize(rd, defaultBufSize)
}

// Read reads data into p.
// It returns the number of bytes read into p.
// The bytes are taken from at most one Read on the underlying Reader,
// hence n may be less than len(p).
// To read exactly len(p) bytes, use io.ReadFull(b, p).
// If the underlying Reader can return a non-zero count with io.EOF,
// then this Read method can do so as well; see the [io.Reader] docs.
func (b *ReadSeeker) Read(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		if b.buffered() > 0 {
			return 0, nil
		}
		return 0, b.readErr()
	}

	if b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}

		if len(p) >= len(b.buf) {
			// large read, empty buffer
			// read directly into p to avoid copy
			if n, b.err = b.rd.Read(p); n < 0 {
				panic(errNegativeRead)
			}

			b.pos += int64(n)
			return n, b.readErr()
		}

		// one read
		b.pos += int64(b.r)
		b.r, b.w = 0, 0
		if n, b.err = b.rd.Read(b.buf); n < 0 {
			panic(errNegativeRead)
		} else if n == 0 {
			return 0, b.readErr()
		}

		b.w += n
	}

	// copy as much as possible
	n = copy(p, b.buf[b.r:b.w])
	b.r += n
	return n, nil
}

func (b *ReadSeeker) Seek(offset int64, whence int) (int64, error) {
	// stream.Seek() implementation makes heavy use of seeking with offset 0
	// to obtain the current position;
	// let's optimize for it
	if offset == 0 && whence == io.SeekCurrent {
		return b.position(), nil
	}
	// when seeking from the end,
	// the absolute position isn't known by ReadSeeker so the current buffer cannot be used.
	// Seeking cannot be avoided.
	if whence == io.SeekEnd {
		return b.seek(offset, whence)
	}

	// Calculate the absolute offset.
	abs := offset
	if whence == io.SeekCurrent {
		abs += b.position()
	}

	// Check if the offset is within buf.
	if abs >= b.pos && abs < b.pos+int64(b.w) {
		b.r = int(abs - b.pos)
		return abs, nil
	}

	return b.seek(abs, io.SeekStart)
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

// position returns the absolute read offset.
func (b *ReadSeeker) position() int64 {
	return b.pos + int64(b.r)
}

func (b *ReadSeeker) reset(buf []byte, r io.ReadSeeker) {
	*b = ReadSeeker{
		buf: buf,
		rd:  r,
	}
}

func (b *ReadSeeker) seek(offset int64, whence int) (int64, error) {
	b.r, b.w = 0, 0
	return b.rd.Seek(offset, whence)
}
