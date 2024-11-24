package flac

import (
	"crypto/md5"
	"hash"
	"io"

	"github.com/icza/bitio"
	"github.com/pchchv/flac/meta"
)

// Encoder represents a FLAC encoder.
type Encoder struct {
	// FLAC stream of encoder.
	*Stream
	// Underlying io.Writer or io.WriteCloser to the output stream.
	w io.Writer
	// Minimum and maximum block size (in samples) of frames written by encoder.
	blockSizeMin, blockSizeMax uint16
	// Minimum and maximum frame size (in bytes) of frames written by encoder.
	frameSizeMin, frameSizeMax uint32
	// MD5 running hash of unencoded audio samples.
	md5sum hash.Hash
	// Total number of samples (per channel) written by encoder.
	nsamples uint64
	// Current frame number if block size is fixed,
	// and the first sample number of the current frame otherwise.
	curNum uint64
}

// NewEncoder returns a new FLAC encoder for the
// given metadata StreamInfo block and optional metadata blocks.
func NewEncoder(w io.Writer, info *meta.StreamInfo, blocks ...*meta.Block) (*Encoder, error) {
	// store FLAC signature
	enc := &Encoder{
		Stream: &Stream{
			Info:   info,
			Blocks: blocks,
		},
		w:      w,
		md5sum: md5.New(),
	}

	bw := bitio.NewWriter(w)
	if _, err := bw.Write(flacSignature); err != nil {
		return nil, err
	}

	// encode metadata blocks
	if err := encodeStreamInfo(bw, info, len(blocks) == 0); err != nil {
		return nil, err
	}

	for i, block := range blocks {
		if err := encodeBlock(bw, block, i == len(blocks)-1); err != nil {
			return nil, err
		}
	}

	// flush pending writes of metadata blocks
	if _, err := bw.Align(); err != nil {
		return nil, err
	}

	// return encoder to be used for encoding audio samples
	return enc, nil
}
