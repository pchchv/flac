package flac

import (
	"fmt"

	"github.com/icza/bitio"
	"github.com/pchchv/flac/frame"
	"github.com/pchchv/flac/internal/bits"
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

// encodeRiceResidual encodes a Rice residual (error signal).
func encodeRiceResidual(bw *bitio.Writer, k uint, residual int32) error {
	// ZigZag encode
	folded := bits.EncodeZigZag(residual)

	// unfold into low- and high
	lowMask := ^uint32(0) >> (32 - k) // lower k bits
	highMask := ^uint32(0) << k       // upper bits
	high := (folded & highMask) >> k
	low := folded & lowMask
	// write unary encoded most significant bits
	if err := bits.WriteUnary(bw, uint64(high)); err != nil {
		return err
	}

	// write binary encoded least significant bits
	if err := bw.WriteBits(uint64(low), uint8(k)); err != nil {
		return err
	}

	return nil
}

// encodeSubframeHeader encodes the given subframe header, writing to bw.
func encodeSubframeHeader(bw *bitio.Writer, subHdr frame.SubHeader) error {
	// zero bit padding, to prevent sync-fooling string of 1s
	if err := bw.WriteBits(0x0, 1); err != nil {
		return err
	}

	// subframe type:
	//     000000 : SUBFRAME_CONSTANT
	//     000001 : SUBFRAME_VERBATIM
	//     00001x : reserved
	//     0001xx : reserved
	//     001xxx : if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
	//     01xxxx : reserved
	//     1xxxxx : SUBFRAME_LPC, xxxxx=order-1
	var ubits uint64
	switch subHdr.Pred {
	case frame.PredConstant:
		// 000000 : SUBFRAME_CONSTANT
		ubits = 0x00
	case frame.PredVerbatim:
		// 000001 : SUBFRAME_VERBATIM
		ubits = 0x01
	case frame.PredFixed:
		// 001xxx : if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
		ubits = 0x08 | uint64(subHdr.Order)
	case frame.PredFIR:
		// 1xxxxx : SUBFRAME_LPC, xxxxx=order-1
		ubits = 0x20 | uint64(subHdr.Order-1)
	}
	if err := bw.WriteBits(ubits, 6); err != nil {
		return err
	}

	// <1+k> 'Wasted bits-per-sample' flag:
	//     0 : no wasted bits-per-sample in source subblock, k=0
	//     1 : k wasted bits-per-sample in source subblock, k-1 follows, unary coded; e.g. k=3 => 001 follows, k=7 => 0000001 follows
	hasWastedBits := subHdr.Wasted > 0
	if err := bw.WriteBool(hasWastedBits); err != nil {
		return err
	}

	if hasWastedBits {
		if err := bits.WriteUnary(bw, uint64(subHdr.Wasted-1)); err != nil {
			return err
		}
	}

	return nil
}

// encodeRicePart encodes a Rice partition of residuals from the subframe,
// using a Rice parameter of the specified size in bits.
func encodeRicePart(bw *bitio.Writer, subframe *frame.Subframe, paramSize uint, residuals []int32) error {
	// 4 bits: Partition order
	riceSubframe := subframe.RiceSubframe
	if err := bw.WriteBits(uint64(riceSubframe.PartOrder), 4); err != nil {
		return err
	}

	// parse Rice partitions; in total 2^partOrder partitions
	partOrder := riceSubframe.PartOrder
	nparts := 1 << partOrder
	curResidualIndex := 0
	for i := range riceSubframe.Partitions {
		partition := &riceSubframe.Partitions[i]
		// (4 or 5) bits: Rice parameter
		param := partition.Param
		if err := bw.WriteBits(uint64(param), uint8(paramSize)); err != nil {
			return err
		}

		// determine the number of Rice encoded samples in the partition
		var nsamples int
		if partOrder == 0 {
			nsamples = subframe.NSamples - subframe.Order
		} else if i != 0 {
			nsamples = subframe.NSamples / nparts
		} else {
			nsamples = subframe.NSamples/nparts - subframe.Order
		}

		if paramSize == 4 && param == 0xF || paramSize == 5 && param == 0x1F {
			// 1111 or 11111: Escape code,
			// meaning the partition is in unencoded binary form using n bits per sample;
			// n follows as a 5-bit number
			if err := bw.WriteBits(uint64(partition.EscapedBitsPerSample), 5); err != nil {
				return err
			}
			for j := 0; j < nsamples; j++ {
				// From section 9.2.7.1.
				// Escaped partition:
				// The residual samples themselves are stored signed two's complement.
				// I. e. when a partition is escaped and each residual sample is stored with 3 bits,
				// the number -1 is represented as 0b111.
				residual := residuals[curResidualIndex]
				curResidualIndex++
				if err := bw.WriteBits(uint64(residual), uint8(partition.EscapedBitsPerSample)); err != nil {
					return err
				}
			}
			continue
		}

		// encode the Rice residuals of the partition
		for j := 0; j < nsamples; j++ {
			residual := residuals[curResidualIndex]
			curResidualIndex++
			if err := encodeRiceResidual(bw, param, residual); err != nil {
				return err
			}
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