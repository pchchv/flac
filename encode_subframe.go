package flac

import (
	"fmt"

	"github.com/icza/bitio"
	"github.com/pchchv/flac/frame"
)

// encodeConstantSamples stores the given constant sample, writing to bw.
func encodeConstantSamples(bw *bitio.Writer, hdr frame.Header, subframe *frame.Subframe, bps uint) error {
	samples := subframe.Samples
	sample := samples[0]
	for _, s := range samples[1:] {
		if sample != s {
			return fmt.Errorf("constant sample mismatch; expected %v, got %v", sample, s)
		}
	}

	// unencoded constant value of the subblock
	// n = frame's bits-per-sample
	if err := bw.WriteBits(uint64(sample), uint8(bps)); err != nil {
		return err
	}

	return nil
}

// encodeVerbatimSamples stores the given samples verbatim (uncompressed), writing to bw
func encodeVerbatimSamples(bw *bitio.Writer, hdr frame.Header, subframe *frame.Subframe, bps uint) error {
	// unencoded subblock
	// n = frame's bits-per-sample
	// i = frame's blocksize
	samples := subframe.Samples
	if int(hdr.BlockSize) != len(samples) {
		return fmt.Errorf("block size and sample count mismatch; expected %d, got %d", hdr.BlockSize, len(samples))
	}

	for _, sample := range samples {
		if err := bw.WriteBits(uint64(sample), uint8(bps)); err != nil {
			return err
		}
	}

	return nil
}
