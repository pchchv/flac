// Package flac provides access to FLAC (Free Lossless Audio Codec) streams.
package flac

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pchchv/flac/meta"
)

var (
	flacSignature  = []byte("fLaC")                                            // marks the beginning of a FLAC stream
	id3Signature   = []byte("ID3")                                             // marks the beginning of an ID3 stream, used to skip over ID3 data
	ErrNoSeektable = errors.New("stream.searchFromStart: no seektable exists") // seektable has not been created (search in the thread is impossible)
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

// New creates a new Stream for accessing the audio samples of r.
// It reads and parses the FLAC signature and the StreamInfo metadata block,
// but skips all other metadata blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame,
// and call Stream.ParseNext to parse the entire next frame including audio samples.
func New(r io.Reader) (stream *Stream, err error) {
	// verify FLAC signature and parse the StreamInfo metadata block.
	br := bufio.NewReader(r)
	stream = &Stream{r: br}
	block, err := stream.parseStreamInfo()
	if err != nil {
		return nil, err
	}

	// skip the remaining metadata blocks.
	for !block.IsLast {
		block, err = meta.New(br)
		if err != nil && err != meta.ErrReservedType {
			return stream, err
		}

		if err = block.Skip(); err != nil {
			return stream, err
		}
	}

	return stream, nil
}

// Close closes the stream gracefully if the underlying io.Reader also implements the io.Closer interface.
func (stream *Stream) Close() error {
	if closer, ok := stream.r.(io.Closer); ok {
		return closer.Close()
	}

	return nil
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

// searchFromStart searches for the given sample number from
// the start of the seek table and returns
// the last seek point containing the sample number.
// If no seek point contains the sample number,
// the last seek point preceding the sample number is returned.
// If the sample number is lower than the first seek point,
// the first seek point is returned.
func (stream *Stream) searchFromStart(sampleNum uint64) (meta.SeekPoint, error) {
	if len(stream.seekTable.Points) == 0 {
		return meta.SeekPoint{}, ErrNoSeektable
	}

	prev := stream.seekTable.Points[0]
	for _, p := range stream.seekTable.Points {
		if p.SampleNum+uint64(p.NSamples) >= sampleNum {
			return prev, nil
		}
		prev = p
	}

	return prev, nil
}

// Parse creates a new Stream for accessing the metadata blocks and audio samples of r.
// It reads and parses the FLAC signature and all metadata blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame,
// and call Stream.ParseNext to parse the entire next frame including audio samples.
func Parse(r io.Reader) (stream *Stream, err error) {
	// verify FLAC signature and parse the StreamInfo metadata block.
	br := bufio.NewReader(r)
	stream = &Stream{r: br}
	block, err := stream.parseStreamInfo()
	if err != nil {
		return nil, err
	}

	// parse the remaining metadata blocks.
	for !block.IsLast {
		block, err = meta.Parse(br)
		if err != nil {
			if err != meta.ErrReservedType {
				return stream, err
			}
			// skip the body of unknown (reserved) metadata blocks,
			// as stated by the specification.
			if err = block.Skip(); err != nil {
				return stream, err
			}
		}
		stream.Blocks = append(stream.Blocks, block)
	}

	return stream, nil
}

// ParseFile creates a new Stream for accessing the
// metadata blocks and audio samples of path.
// It reads and parses the FLAC signature and all metadata blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame,
// and call Stream.ParseNext to parse the
// entire next frame including audio samples.
//
// Note: Close method of the stream must be called when finished using it.
func ParseFile(path string) (stream *Stream, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	stream, err = Parse(f)
	if err != nil {
		return nil, err
	}

	return
}

// Open creates a new Stream for accessing the audio samples of path.
// It reads and parses the FLAC signature and the StreamInfo metadata block,
// but skips all other metadata blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame,
// and call Stream.ParseNext to parse the entire next frame including audio samples.
//
// Note: The Close method of the stream must be called when finished using it.
func Open(path string) (stream *Stream, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	stream, err = New(f)
	if err != nil {
		return nil, err
	}

	return
}
