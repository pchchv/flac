package meta_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/pchchv/flac"
	"github.com/pchchv/flac/meta"
)

func TestMissingValue(t *testing.T) {
	_, err := flac.ParseFile("testdata/missing-value.flac")
	if err.Error() != `meta.Block.parseVorbisComment: unable to locate '=' in vector "title 2"` {
		t.Fatal(err)
	}
}

func TestParsePicture(t *testing.T) {
	stream, err := flac.ParseFile("testdata/silence.flac")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	want, err := os.ReadFile("testdata/silence.jpg")
	if err != nil {
		t.Fatal(err)
	}

	for _, block := range stream.Blocks {
		if block.Type == meta.TypePicture {
			pic := block.Body.(*meta.Picture)
			got := pic.Data
			if !bytes.Equal(got, want) {
				t.Errorf("picture data differ; expected %v, got %v", want, got)
			}
			break
		}
	}
}
