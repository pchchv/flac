package bufseekio

import (
	"bytes"
	"errors"
	"testing"
)

var expectedErr = errors.New("expected error")

type readAndError struct {
	bytes []byte
}

func (r *readAndError) Read(p []byte) (n int, err error) {
	for i, b := range r.bytes {
		p[i] = b
	}
	return len(r.bytes), expectedErr
}

func (r *readAndError) Seek(offset int64, whence int) (int64, error) {
	panic("not implemented")
}

func TestNewReadSeeker(t *testing.T) {
	buf := bytes.NewReader(make([]byte, 100))
	if rs := NewReadSeeker(buf); len(rs.buf) != defaultBufSize {
		t.Fatalf("want %d got %d", defaultBufSize, len(rs.buf))
	}
}

func TestNewReadSeekerSize(t *testing.T) {
	buf := bytes.NewReader(make([]byte, 100))

	// test custom buffer size
	if rs := NewReadSeekerSize(buf, 20); len(rs.buf) != 20 {
		t.Fatalf("want %d got %d", 20, len(rs.buf))
	}

	// test too small buffer size
	if rs := NewReadSeekerSize(buf, 1); len(rs.buf) != minReadBufferSize {
		t.Fatalf("want %d got %d", minReadBufferSize, len(rs.buf))
	}

	// test reuse existing ReadSeeker
	rs := NewReadSeekerSize(buf, 20)
	if rs2 := NewReadSeekerSize(rs, 5); rs != rs2 {
		t.Fatal("expected ReadSeeker to be reused but got a different ReadSeeker")
	}
}
