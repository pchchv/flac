package frame

import (
	"errors"
	"fmt"

	"github.com/pchchv/flac/internal/bits"
)

// Prediction methods.
const (
	// PredConstant specifies that the subframe contains a constant sound.
	// The audio samples are encoded using run-length encoding.
	// Since every audio sample has the same constant value,
	// a single unencoded audio sample is stored in practice.
	// It is replicated a number of times,
	// as specified by BlockSize in the frame header.
	PredConstant Pred = iota
	// PredVerbatim specifies that the subframe contains unencoded audio samples.
	// Random sound is often stored verbatim,
	// since no prediction method can compress it sufficiently.
	PredVerbatim
	// PredFixed specifies that the subframe contains linear prediction coded audio samples.
	// The coefficients of the prediction polynomial are selected from a fixed set,
	// and can represent 0th through fourth-order polynomials.
	// The prediction order (0 through 4)
	// is stored within the subframe along with the same number of unencoded warm-up samples,
	// which are used to kick start the prediction polynomial.
	// The remainder of the subframe stores encoded residuals (signal errors)
	// which specify the difference between the predicted and the original audio samples.
	PredFixed
	// PredFIR specifies that the subframe contains linear prediction coded audio samples.
	// The coefficients of the prediction polynomial are stored in the subframe,
	// and can represent 0th through 32nd-order polynomials.
	// The prediction order (0 through 32)
	// is stored within the subframe along with the same number of unencoded warm-up samples,
	// which are used to kick start the prediction polynomial.
	// The remainder of the subframe stores encoded residuals (signal errors)
	// which specify the difference between the predicted and the original audio samples.
	PredFIR
)

// FixedCoeffs maps from prediction order to
// the LPC coefficients used in fixed encoding.
//
//	x_0[n] = 0
//	x_1[n] = x[n-1]
//	x_2[n] = 2*x[n-1] - x[n-2]
//	x_3[n] = 3*x[n-1] - 3*x[n-2] + x[n-3]
//	x_4[n] = 4*x[n-1] - 6*x[n-2] + 4*x[n-3] - x[n-4]
var FixedCoeffs = [...][]int32{
	1: {1},
	2: {2, -1},
	3: {3, -3, 1},
	4: {4, -6, 4, -1},
}

// Pred specifies the prediction method used to encode
// the audio samples of a subframe.
type Pred uint8

// ResidualCodingMethod specifies a residual coding method.
type ResidualCodingMethod uint8

// RicePartition is a partition containing
// a subset of the residuals of a subframe.
type RicePartition struct {
	// Rice parameter.
	Param uint
	// Residual sample size in bits-per-sample used by escaped partitions.
	EscapedBitsPerSample uint
}

// RiceSubframe holds rice-coding subframe fields used
// by residual coding methods rice1 and rice2.
type RiceSubframe struct {
	// Partition order used by fixed and FIR linear prediction decoding
	// (for residual coding methods, rice1 and rice2).
	PartOrder int
	// Rice partitions.
	Partitions []RicePartition
}

// SubHeader specifies the prediction method and order of a subframe.
type SubHeader struct {
	// Specifies the prediction method used to encode the audio sample of the subframe.
	Pred Pred
	// Prediction order used by fixed and FIR linear prediction decoding.
	Order int
	// Wasted bits-per-sample.
	Wasted uint
	// Residual coding method used by fixed and FIR linear prediction decoding.
	ResidualCodingMethod ResidualCodingMethod
	// Coefficients' precision in bits used by FIR linear prediction decoding.
	CoeffPrec uint
	// Predictor coefficient shift needed in bits used by FIR linear prediction decoding.
	CoeffShift int32
	// Predictor coefficients used by FIR linear prediction decoding.
	Coeffs []int32
	// Rice-coding subframe fields used by residual coding methods rice1 and rice2; nil if unused.
	RiceSubframe *RiceSubframe
}

// Subframe contains the encoded audio samples from
// one channel of an audio block
// (a part of the audio stream).
type Subframe struct {
	// Subframe header.
	SubHeader
	// Unencoded audio samples.
	// Samples is initially nil, and gets populated by a call to Frame.Parse.
	// Samples is used by decodeFixed and decodeFIR to temporarily store residuals.
	// Before returning they call decodeLPC which decodes the audio samples.
	Samples []int32
	// Number of audio samples in the subframe.
	NSamples int
}

// parseHeader reads and parses the header of a subframe.
func (subframe *Subframe) parseHeader(br *bits.Reader) error {
	// 1 bit: zero-padding.
	x, err := br.Read(1)
	if err != nil {
		return unexpected(err)
	} else if x != 0 {
		return errors.New("frame.Subframe.parseHeader: non-zero padding")
	}

	// 6 bits: Pred.
	if x, err = br.Read(6); err != nil {
		return unexpected(err)
	}

	// The 6 bits are used to specify the prediction method and order as follows:
	//    000000: Constant prediction method.
	//    000001: Verbatim prediction method.
	//    00001x: reserved.
	//    0001xx: reserved.
	//    001xxx:
	//       if (xxx <= 4)
	//          Fixed prediction method; xxx=order
	//       else
	//          reserved.
	//    01xxxx: reserved.
	//    1xxxxx: FIR prediction method; xxxxx=order-1
	switch {
	case x < 1:
		// 000000: Constant prediction method.
		subframe.Pred = PredConstant
	case x < 2:
		// 000001: Verbatim prediction method.
		subframe.Pred = PredVerbatim
	case x < 8:
		// 00001x: reserved.
		// 0001xx: reserved.
		return fmt.Errorf("frame.Subframe.parseHeader: reserved prediction method bit pattern (%06b)", x)
	case x < 16:
		// 001xxx:
		//    if (xxx <= 4)
		//       Fixed prediction method; xxx=order
		//    else
		//       reserved.
		order := int(x & 0x07)
		if order > 4 {
			return fmt.Errorf("frame.Subframe.parseHeader: reserved prediction method bit pattern (%06b)", x)
		}
		subframe.Pred = PredFixed
		subframe.Order = order
	case x < 32:
		// 01xxxx: reserved.
		return fmt.Errorf("frame.Subframe.parseHeader: reserved prediction method bit pattern (%06b)", x)
	default:
		// 1xxxxx: FIR prediction method; xxxxx=order-1
		subframe.Pred = PredFIR
		subframe.Order = int(x&0x1F) + 1
	}

	// 1 bit: hasWastedBits.
	if x, err = br.Read(1); err != nil {
		return unexpected(err)
	} else if x != 0 {
		// k wasted bits-per-sample in source subblock, k-1 follows, unary coded;
		// e.g. k=3 => 001 follows, k=7 => 0000001 follows.
		if x, err = br.ReadUnary(); err != nil {
			return unexpected(err)
		}
		subframe.Wasted = uint(x) + 1
	}

	return nil
}

// decodeConstant reads an unencoded audio sample of the subframe.
// Each sample of the subframe has this constant value.
// The constant encoding can be thought of as run-length encoding.
func (subframe *Subframe) decodeConstant(br *bits.Reader, bps uint) error {
	// (bits-per-sample) bits: Unencoded constant value of the subblock.
	x, err := br.Read(bps)
	if err != nil {
		return unexpected(err)
	}

	// Each sample of the subframe has the same constant value.
	sample := signExtend(x, bps)
	for i := 0; i < subframe.NSamples; i++ {
		subframe.Samples = append(subframe.Samples, sample)
	}

	return nil
}

// decodeVerbatim reads the unencoded audio samples of the subframe.
func (subframe *Subframe) decodeVerbatim(br *bits.Reader, bps uint) error {
	// Parse the unencoded audio samples of the subframe.
	for i := 0; i < subframe.NSamples; i++ {
		// (bits-per-sample) bits: Unencoded constant value of the subblock.
		x, err := br.Read(bps)
		if err != nil {
			return unexpected(err)
		}

		sample := signExtend(x, bps)
		subframe.Samples = append(subframe.Samples, sample)
	}

	return nil
}

// decodeRiceResidual decodes and returns
// a Rice encoded residual (error signal).
func (subframe *Subframe) decodeRiceResidual(br *bits.Reader, k uint) (int32, error) {
	// Read unary encoded most significant bits.
	high, err := br.ReadUnary()
	if err != nil {
		return 0, unexpected(err)
	}

	// Read binary encoded least significant bits.
	low, err := br.Read(k)
	if err != nil {
		return 0, unexpected(err)
	}
	folded := uint32(high<<k | low)

	// ZigZag decode.
	residual := bits.DecodeZigZag(folded)
	return residual, nil
}

// decodeRicePart decodes a Rice partition of encoded residuals from the subframe,
// using a Rice parameter of the specified size in bits.
func (subframe *Subframe) decodeRicePart(br *bits.Reader, paramSize uint) error {
	// 4 bits: Partition order.
	x, err := br.Read(4)
	if err != nil {
		return unexpected(err)
	}

	partOrder := int(x)
	riceSubframe := &RiceSubframe{
		PartOrder: partOrder,
	}
	subframe.RiceSubframe = riceSubframe

	// parse Rice partitions; in total 2^partOrder partitions.
	nparts := 1 << partOrder
	partitions := make([]RicePartition, nparts)
	riceSubframe.Partitions = partitions
	for i := 0; i < nparts; i++ {
		partition := &partitions[i]
		// (4 or 5) bits: Rice parameter.
		x, err = br.Read(paramSize)
		if err != nil {
			return unexpected(err)
		}

		param := uint(x)
		partition.Param = param

		// determine the number of Rice encoded samples in the partition.
		var nsamples int
		if partOrder == 0 {
			nsamples = subframe.NSamples - subframe.Order
		} else if i != 0 {
			nsamples = subframe.NSamples / nparts
		} else {
			nsamples = subframe.NSamples/nparts - subframe.Order
		}

		if paramSize == 4 && param == 0xF || paramSize == 5 && param == 0x1F {
			// 1111 or 11111: Escape code, meaning the partition is in unencoded
			// binary form using n bits per sample; n follows as a 5-bit number.
			x, err := br.Read(5)
			if err != nil {
				return unexpected(err)
			}

			n := uint(x)
			partition.EscapedBitsPerSample = n
			for j := 0; j < nsamples; j++ {
				sample, err := br.Read(n)
				if err != nil {
					return unexpected(err)
				}
				// from section 9.2.7.1.  Escaped partition:
				//
				// The residual samples themselves are stored signed two's complement.
				// i. e., when a partition is escaped and each residual sample is stored with 3 bits,
				// the number -1 is represented as 0b111.
				subframe.Samples = append(subframe.Samples, int32(bits.IntN(sample, n)))
			}
			continue
		}

		// decode the Rice encoded residuals of the partition.
		for j := 0; j < nsamples; j++ {
			residual, err := subframe.decodeRiceResidual(br, param)
			if err != nil {
				return err
			}
			subframe.Samples = append(subframe.Samples, residual)
		}
	}

	return nil
}

// decodeResiduals decodes the encoded residuals
// (prediction method error signals) of the subframe.
func (subframe *Subframe) decodeResiduals(br *bits.Reader) error {
	// 2 bits: Residual coding method.
	x, err := br.Read(2)
	if err != nil {
		return unexpected(err)
	}

	residualCodingMethod := ResidualCodingMethod(x)
	subframe.ResidualCodingMethod = residualCodingMethod
	// The 2 bits are used to specify the residual coding method as follows:
	//    00: Rice coding with a 4-bit Rice parameter.
	//    01: Rice coding with a 5-bit Rice parameter.
	//    10: reserved.
	//    11: reserved.
	switch residualCodingMethod {
	case 0x0:
		return subframe.decodeRicePart(br, 4)
	case 0x1:
		return subframe.decodeRicePart(br, 5)
	default:
		return fmt.Errorf("frame.Subframe.decodeResiduals: reserved residual coding method bit pattern (%02b)", uint8(residualCodingMethod))
	}
}

// decodeLPC decodes linear prediction coded audio samples,
// using the coefficients of a given polynomial,
// a couple of unencoded warm-up samples,
// and the signal errors of the prediction as specified by the residuals.
func (subframe *Subframe) decodeLPC(coeffs []int32, shift int32) error {
	if len(coeffs) != subframe.Order {
		return fmt.Errorf("frame.Subframe.decodeLPC: prediction order (%d) differs from number of coefficients (%d)", subframe.Order, len(coeffs))
	}

	if shift < 0 {
		return fmt.Errorf("frame.Subframe.decodeLPC: invalid negative shift")
	}

	if subframe.NSamples != len(subframe.Samples) {
		return fmt.Errorf("frame.Subframe.decodeLPC: subframe sample count mismatch; expected %d, got %d", subframe.NSamples, len(subframe.Samples))
	}

	for i := subframe.Order; i < subframe.NSamples; i++ {
		var sample int64
		for j, c := range coeffs {
			sample += int64(c) * int64(subframe.Samples[i-j-1])
		}
		subframe.Samples[i] += int32(sample >> uint(shift))
	}

	return nil
}

// decodeFixed decodes the linear prediction coded samples of the subframe,
// using a fixed set of predefined polynomial coefficients.
func (subframe *Subframe) decodeFixed(br *bits.Reader, bps uint) error {
	// parse unencoded warm-up samples
	for i := 0; i < subframe.Order; i++ {
		// (bits-per-sample) bits: Unencoded warm-up sample
		x, err := br.Read(bps)
		if err != nil {
			return unexpected(err)
		}
		sample := signExtend(x, bps)
		subframe.Samples = append(subframe.Samples, sample)
	}

	// decode subframe residuals
	if err := subframe.decodeResiduals(br); err != nil {
		return err
	}

	// Predict the audio samples of the subframe using
	// a polynomial with predefined coefficients of a given order.
	// Correct signal errors using the decoded residuals.
	const shift = 0
	return subframe.decodeLPC(FixedCoeffs[subframe.Order], shift)
}

// signExtend interprets x as a signed n-bit integer value
// and sign extends it to 32 bits.
func signExtend(x uint64, n uint) int32 {
	// x is signed if its most significant bit is set.
	if x&(1<<(n-1)) != 0 {
		// sign extend x
		return int32(x | ^uint64(0)<<n)
	}

	return int32(x)
}
