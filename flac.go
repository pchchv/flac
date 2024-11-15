// Package flac provides access to FLAC (Free Lossless Audio Codec) streams.
package flac

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/pchchv/flac/meta"
)

var (
	flacSignature = []byte("fLaC") // marks the beginning of a FLAC stream
	id3Signature  = []byte("ID3")  // marks the beginning of an ID3 stream, used to skip over ID3 data

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

// skipID3v2 skips ID3v2 data prepended to flac files.
func (stream *Stream) skipID3v2() error {
	r := bufio.NewReader(stream.r)
	// discard unnecessary data from the ID3v2 header.
	if _, err := r.Discard(2); err != nil {
		return err
	}

	// read the size from the ID3v2 header.
	var sizeBuf [4]byte
	if _, err := r.Read(sizeBuf[:]); err != nil {
		return err
	}

	// size is encoded as a synchsafe integer.
	size := int(sizeBuf[0])<<21 | int(sizeBuf[1])<<14 | int(sizeBuf[2])<<7 | int(sizeBuf[3])
	_, err := r.Discard(size)
	return err
}

// parseStreamInfo verifies the signature which marks the beginning of a FLAC stream,
// and parses the StreamInfo metadata block.
// It returns a boolean value which specifies if the
// StreamInfo block was the last metadata block of the FLAC stream.
func (stream *Stream) parseStreamInfo() (block *meta.Block, err error) {
	// verify FLAC signature.
	r := stream.r
	var buf [4]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return block, err
	}

	// skip prepended ID3v2 data.
	if bytes.Equal(buf[:3], id3Signature) {
		if err := stream.skipID3v2(); err != nil {
			return block, err
		}

		// second attempt at verifying signature.
		if _, err = io.ReadFull(r, buf[:]); err != nil {
			return block, err
		}
	}

	if !bytes.Equal(buf[:], flacSignature) {
		return block, fmt.Errorf("flac.parseStreamInfo: invalid FLAC signature; expected %q, got %q", flacSignature, buf)
	}

	// parse StreamInfo metadata block.
	block, err = meta.Parse(r)
	if err != nil {
		return block, err
	}

	si, ok := block.Body.(*meta.StreamInfo)
	if !ok {
		return block, fmt.Errorf("flac.parseStreamInfo: incorrect type of first metadata block; expected *meta.StreamInfo, got %T", block.Body)
	}

	stream.Info = si
	return block, nil
}
