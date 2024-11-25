package flac_test

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/pchchv/flac"
)

func TestSkipID3v2(t *testing.T) {
	if _, err := flac.ParseFile("testdata/id3.flac"); err != nil {
		t.Fatal(err)
	}
}

func TestSeek(t *testing.T) {
	f, err := os.Open("testdata/172960.flac")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	// seek Table:
	// {SampleNum:0 Offset:8283 NSamples:4096}
	// {SampleNum:4096 Offset:17777 NSamples:4096}
	// {SampleNum:8192 Offset:27141 NSamples:4096}
	// {SampleNum:12288 Offset:36665 NSamples:4096}
	// {SampleNum:16384 Offset:46179 NSamples:4096}
	// {SampleNum:20480 Offset:55341 NSamples:4096}
	// {SampleNum:24576 Offset:64690 NSamples:4096}
	// {SampleNum:28672 Offset:74269 NSamples:4096}
	// {SampleNum:32768 Offset:81984 NSamples:4096}
	// {SampleNum:36864 Offset:86656 NSamples:4096}
	// {SampleNum:40960 Offset:89596 NSamples:2723}

	testPos := []struct {
		seek     uint64
		expected uint64
		err      string
	}{
		{seek: 0, expected: 0},
		{seek: 9000, expected: 8192},
		{seek: 0, expected: 0},
		{seek: 8000, expected: 4096},
		{seek: 0, expected: 0},
		{seek: 50000, expected: 0, err: "unable to seek to sample number 50000"},
		{seek: 100, expected: 0},
		{seek: 8192, expected: 8192},
		{seek: 8191, expected: 4096},
		{seek: 40960 + 2723, expected: 0, err: "unable to seek to sample number 43683"},
	}

	stream, err := flac.NewSeek(f)
	if err != nil {
		t.Fatal(err)
	}

	for i, pos := range testPos {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			p, err := stream.Seek(pos.seek)
			if err != nil {
				if err.Error() != pos.err {
					t.Fatal(err)
				}
			}

			if p != pos.expected {
				t.Fatalf("pos %d does not equal %d", p, pos.expected)
			}

			if _, err = stream.ParseNext(); err != nil && err != io.EOF {
				t.Fatal(err)
			}
		})
	}
}
