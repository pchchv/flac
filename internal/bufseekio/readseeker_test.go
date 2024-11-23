package bufseekio

import (
	"bytes"
	"testing"
)

func TestNewReadSeeker(t *testing.T) {
	buf := bytes.NewReader(make([]byte, 100))
	if rs := NewReadSeeker(buf); len(rs.buf) != defaultBufSize {
		t.Fatalf("want %d got %d", defaultBufSize, len(rs.buf))
	}
}
