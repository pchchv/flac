package bufseekio

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

var expectedErr = errors.New("expected error")

type seekRecord struct {
	offset int64
	whence int
}

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

func TestReadSeeker_Read(t *testing.T) {
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}

	rs := NewReadSeekerSize(bytes.NewReader(data), 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	// test small read
	got := make([]byte, 5)
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{0, 1, 2, 3, 4}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{0, 1, 2, 3, 4}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 5 {
		t.Fatalf("want %d got %d, err=%v", 5, p, err)
	}

	// test big read with initially filled buffer
	got = make([]byte, 25)
	if n, err := rs.Read(got); err != nil || n != 15 || !reflect.DeepEqual(got, []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 15, n, []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, got, err)
	}

	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 20 {
		t.Fatalf("want %d got %d, err=%v", 20, p, err)
	}

	// test big read with initially empty buffer
	if n, err := rs.Read(got); err != nil || n != 25 || !reflect.DeepEqual(got, []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 25, n, []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44}, got, err)
	}

	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 45 {
		t.Fatalf("want %d got %d, err=%v", 45, p, err)
	}

	// test EOF
	if p, err := rs.Seek(98, io.SeekStart); err != nil || p != 98 {
		t.Fatalf("want %d got %d, err=%v", 98, p, err)
	}

	got = make([]byte, 5)
	if n, err := rs.Read(got); err != nil || n != 2 || !reflect.DeepEqual(got, []byte{98, 99, 0, 0, 0}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 2, n, []byte{98, 99, 0, 0, 0}, got, err)
	}

	if n, err := rs.Read(got); err != io.EOF || n != 0 {
		t.Fatalf("want n read %d got %d, err=%v", 0, n, err)
	}

	// test source that returns bytes and an error at the same time
	rs = NewReadSeekerSize(&readAndError{bytes: []byte{2, 3, 5}}, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	got = make([]byte, 5)
	if n, err := rs.Read(got); err != nil || n != 3 || !reflect.DeepEqual(got, []byte{2, 3, 5, 0, 0}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 3, n, []byte{2, 3, 5, 0, 0}, got, err)
	}

	if n, err := rs.Read(got); err != expectedErr || n != 0 {
		t.Fatalf("want n read %d got %d, want error %v, got %v", 0, n, expectedErr, err)
	}

	// test read nothing with an empty buffer and a queued error
	rs = NewReadSeekerSize(&readAndError{bytes: []byte{2, 3, 5}}, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	got = make([]byte, 3)
	if n, err := rs.Read(got); err != nil || n != 3 || !reflect.DeepEqual(got, []byte{2, 3, 5}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 3, n, []byte{2, 3, 5}, got, err)
	}

	if n, err := rs.Read(nil); err != expectedErr || n != 0 {
		t.Fatalf("want n read %d got %d, want error %v, got %v", 0, n, expectedErr, err)
	}

	if n, err := rs.Read(nil); err != nil || n != 0 {
		t.Fatalf("want n read %d got %d, err=%v", 0, n, err)
	}

	// test read nothing with a non-empty buffer and a queued error
	rs = NewReadSeekerSize(&readAndError{bytes: []byte{2, 3, 5}}, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	got = make([]byte, 1)
	if n, err := rs.Read(got); err != nil || n != 1 || !reflect.DeepEqual(got, []byte{2}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 1, n, []byte{}, got, err)
	}

	if n, err := rs.Read(nil); err != nil || n != 0 {
		t.Fatalf("want n read %d got %d, err=%v", 0, n, err)
	}
}
