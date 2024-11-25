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

// getLPCResiduals returns the residuals
// (signal errors of the prediction)
// between the given audio samples and the LPC predicted audio samples,
// using the coefficients of a given polynomial,
// and a couple (order of polynomial;
// i.e. len(coeffs)) of unencoded warm-up samples.
func getLPCResiduals(subframe *frame.Subframe, coeffs []int32, shift int32) ([]int32, error) {
	if len(coeffs) != subframe.Order {
		return nil, fmt.Errorf("getLPCResiduals: prediction order (%d) differs from number of coefficients (%d)", subframe.Order, len(coeffs))
	}

	if shift < 0 {
		return nil, fmt.Errorf("getLPCResiduals: invalid negative shift")
	}

	if subframe.NSamples != len(subframe.Samples) {
		return nil, fmt.Errorf("getLPCResiduals: subframe sample count mismatch; expected %d, got %d", subframe.NSamples, len(subframe.Samples))
	}

	var residuals []int32
	for i := subframe.Order; i < subframe.NSamples; i++ {
		var sample int64
		for j, c := range coeffs {
			sample += int64(c) * int64(subframe.Samples[i-j-1])
		}
		residual := subframe.Samples[i] - int32(sample>>uint(shift))
		residuals = append(residuals, residual)
	}

	return residuals, nil
}
