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

// Close closes the underlying io.Writer of the encoder and flushes any pending writes.
// If the io.Writer implements io.Seeker,
// the encoder will update the StreamInfo metadata block with the
// MD5 checksum of the unencoded audio samples,
// the number of samples,
// and the minimum and maximum frame size and block size.
func (enc *Encoder) Close() error {
	// update StreamInfo metadata block
	if ws, ok := enc.w.(io.WriteSeeker); ok {
		if _, err := ws.Seek(int64(len(flacSignature)), io.SeekStart); err != nil {
			return err
		}
		// update minimum and maximum block size (in samples) of FLAC stream
		enc.Info.BlockSizeMin = enc.blockSizeMin
		enc.Info.BlockSizeMax = enc.blockSizeMax
		// update minimum and maximum frame size (in bytes) of FLAC stream
		enc.Info.FrameSizeMin = enc.frameSizeMin
		enc.Info.FrameSizeMax = enc.frameSizeMax
		// update total number of samples (per channel) of FLAC stream
		enc.Info.NSamples = enc.nsamples
		// update MD5 checksum of the unencoded audio samples
		sum := enc.md5sum.Sum(nil)
		for i := range sum {
			enc.Info.MD5sum[i] = sum[i]
		}

		bw := bitio.NewWriter(ws)
		// write updated StreamInfo metadata block to output stream
		if err := encodeStreamInfo(bw, enc.Info, len(enc.Blocks) == 0); err != nil {
			return err
		}

		if _, err := bw.Align(); err != nil {
			return err
		}
	}

	if closer, ok := enc.w.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
