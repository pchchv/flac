package meta_test

import (
	"testing"

	"github.com/pchchv/flac"
)

func TestMissingValue(t *testing.T) {
	_, err := flac.ParseFile("testdata/missing-value.flac")
	if err.Error() != `meta.Block.parseVorbisComment: unable to locate '=' in vector "title 2"` {
		t.Fatal(err)
	}
}
