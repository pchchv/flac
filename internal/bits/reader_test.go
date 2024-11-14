package bits

import (
	"bytes"
	"io"
	"testing"
)

func TestReadEOF(t *testing.T) {
	tests := []struct {
		data []byte
		n    uint
		err  error
	}{
		{[]byte{0xFF}, 8, nil},
		{[]byte{0xFF}, 2, nil},
		{[]byte{0xFF}, 9, io.ErrUnexpectedEOF},
		{[]byte{}, 1, io.EOF},
		{[]byte{0xFF, 0xFF}, 16, nil},
		{[]byte{0xFF, 0xFF}, 17, io.ErrUnexpectedEOF},
	}

	for i, test := range tests {
		r := NewReader(bytes.NewReader(test.data))
		if _, err := r.Read(test.n); err != test.err {
			t.Errorf("i=%d; Reading %d from %v, expected err=%s, got err=%s", i, test.n, test.data, test.err, err)
		}
	}
}
