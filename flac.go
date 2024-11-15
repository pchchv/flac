// Package flac provides access to FLAC (Free Lossless Audio Codec) streams.
package flac

import (
	"io"

	"github.com/pchchv/flac/meta"
)

// Stream contains the metadata blocks and
// provides access to the audio frames of a FLAC stream.
type Stream struct {
	// The StreamInfo metadata block describes
	// the basic properties of the FLAC audio stream.
	Info *meta.StreamInfo
	// Zero or more metadata blocks.
	Blocks []*meta.Block
	// seekTable contains one or
	// more pre-calculated audio frame seek points of the stream;
	// nil if uninitialized.
	seekTable *meta.SeekTable
	// seekTableSize determines how many seek points
	// the seekTable should have if the
	// flac file does not include one in the metadata.
	seekTableSize int
	// dataStart is the offset of the
	// first frame header since SeekPoint.Offset
	// is relative to this position.
	dataStart int64
	// Underlying io.Reader, or io.ReadCloser.
	r io.Reader
}
