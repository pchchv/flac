package meta

// A SeekPoint specifies the byte offset and
// initial sample number of a given target frame.
type SeekPoint struct {
	// Sample number of the first sample in the target frame,
	// or 0xFFFFFFFFFFFFFFFF for a placeholder point.
	SampleNum uint64
	// Offset in bytes from the first byte of
	// the first frame header to the first byte of
	// the target frame's header.
	Offset uint64
	// Number of samples in the target frame.
	NSamples uint16
}

// SeekTable contains one or more pre-calculated audio frame seek points.
type SeekTable struct {
	Points []SeekPoint // one or more seek points
}
