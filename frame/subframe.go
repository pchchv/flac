package frame

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
