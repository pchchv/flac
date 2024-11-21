// Package frame implements access to FLAC audio frames.
// FLAC encoders divide the audio stream into blocks through a process called blocking.
// A block contains uncoded audio samples from all channels in a short period of time.
// Each audio block is divided into sub-blocks, one per channel.
// There is often a correlation between the left and right channels of stereo audio.
// Using inter-channel decorrelation,
// it is possible to store only one of the channels and the difference between them,
// or store the average of the channels and their difference.
// The encoder decorrelates audio samples as follows:
//
//	mid = (left + right)/2 // average of the channels
//	side = left - right    // difference between the channels
//
// Blocks are encoded using different prediction methods and stored in frames.
// Blocks and sub-blocks contain unencoded audio samples,
// while frames and sub-frames contain encoded audio samples.
// A FLAC stream contains one or more audio frames.
package frame

import (
	"io"

	"github.com/pchchv/flac/internal/bits"
	"github.com/pchchv/flac/internal/hashutil"
)

// Channel assignments.
// Used abbreviations:
//
//	C:   center (directly in front)
//	R:   right (standard stereo)
//	Sr:  side right (directly to the right)
//	Rs:  right surround (back right)
//	Cs:  center surround (rear center)
//	Ls:  left surround (back left)
//	Sl:  side left (directly to the left)
//	L:   left (standard stereo)
//	Lfe: low-frequency effect (placed according to room acoustics)
//
// The first 6 channel constants follow the SMPTE/ITU-R channel order:
//
//	L R C Lfe Ls Rs
const (
	ChannelsMono           Channels = iota // 1 channel: mono.
	ChannelsLR                             // 2 channels: left, right.
	ChannelsLRC                            // 3 channels: left, right, center.
	ChannelsLRLsRs                         // 4 channels: left, right, left surround, right surround.
	ChannelsLRCLsRs                        // 5 channels: left, right, center, left surround, right surround.
	ChannelsLRCLfeLsRs                     // 6 channels: left, right, center, LFE, left surround, right surround.
	ChannelsLRCLfeCsSlSr                   // 7 channels: left, right, center, LFE, center surround, side left, side right.
	ChannelsLRCLfeLsRsSlSr                 // 8 channels: left, right, center, LFE, left surround, right surround, side left, side right.
	ChannelsLeftSide                       // 2 channels: left, side; using inter-channel decorrelation.
	ChannelsSideRight                      // 2 channels: side, right; using inter-channel decorrelation.
	ChannelsMidSide                        // 2 channels: mid, side; using inter-channel decorrelation.
)

// nChannels specifies the number of channels used by each channel assignment.
var nChannels = [...]int{
	ChannelsMono:           1,
	ChannelsLR:             2,
	ChannelsLRC:            3,
	ChannelsLRLsRs:         4,
	ChannelsLRCLsRs:        5,
	ChannelsLRCLfeLsRs:     6,
	ChannelsLRCLfeCsSlSr:   7,
	ChannelsLRCLfeLsRsSlSr: 8,
	ChannelsLeftSide:       2,
	ChannelsSideRight:      2,
	ChannelsMidSide:        2,
}

// Channels specifies the number of channels (subframes) that exist in a frame,
// their order and possible inter-channel decorrelation.
type Channels uint8

// Count returns the number of channels (subframes) used by
// the provided channel assignment.
func (channels Channels) Count() int {
	return nChannels[channels]
}

// Header contains the basic properties of an audio frame,
// such as its sample rate and channel count.
// To facilitate random access decoding each frame header starts with a sync-code.
// This allows the decoder to synchronize and locate the start of a frame header.
type Header struct {
	// Specifies if the block size is fixed or variable.
	HasFixedBlockSize bool
	// Block size in inter-channel samples,
	// i.e. the number of audio samples in each subframe.
	BlockSize uint16
	// Sample rate in Hz; a 0 value implies unknown,
	// get sample rate from StreamInfo.
	SampleRate uint32
	// Specifies the number of channels (subframes) that exist in the frame,
	// their order and possible inter-channel decorrelation.
	Channels Channels
	// Sample size in bits-per-sample;
	// a 0 value implies unknown, get sample size from StreamInfo.
	BitsPerSample uint8
	// Specifies the frame number if the block size is fixed,
	// and the first sample number in the frame otherwise.
	// When using fixed block size,
	// the first sample number in the frame can be derived
	// by multiplying the frame number with the block size (in samples).
	Num uint64
}

// Frame contains the header and subframes of an audio frame.
// It holds the encoded samples from a block (a part) of the audio stream.
// Each subframe holding the samples from one of its channel.
type Frame struct {
	// Audio frame header.
	Header
	// One subframe per channel, containing encoded audio samples.
	Subframes []*Subframe
	// CRC-16 hash sum, calculated by read operations on hr.
	crc hashutil.Hash16
	// A bit reader, wrapping read operations to hr.
	br *bits.Reader
	// A CRC-16 hash reader, wrapping read operations to r.
	hr io.Reader
	// Underlying io.Reader.
	r io.Reader
}

// Correlate reverts any inter-channel decorrelation between the samples of the subframes.
// An encoder decorrelates audio samples as follows:
//
//	mid = (left + right)/2
//	side = left - right
func (frame *Frame) Correlate() {
	switch frame.Channels {
	case ChannelsLeftSide:
		// 2 channels: left, side; using inter-channel decorrelation.
		left := frame.Subframes[0].Samples
		side := frame.Subframes[1].Samples
		for i := range side {
			// right = left - side
			side[i] = left[i] - side[i]
		}
	case ChannelsSideRight:
		// 2 channels: side, right; using inter-channel decorrelation.
		side := frame.Subframes[0].Samples
		right := frame.Subframes[1].Samples
		for i := range side {
			// left = right + side
			side[i] = right[i] + side[i]
		}
	case ChannelsMidSide:
		// 2 channels: mid, side; using inter-channel decorrelation.
		mid := frame.Subframes[0].Samples
		side := frame.Subframes[1].Samples
		for i := range side {
			// left = (2*mid + side)/2
			// right = (2*mid - side)/2
			m := mid[i]
			s := side[i]
			m *= 2
			// Notice that the integer division in mid = (left + right)/2 discards
			// the least significant bit. It can be reconstructed however, since a
			// sum A+B and a difference A-B has the same least significant bit.
			//
			// ref: Data Compression: The Complete Reference (ch. 7, Decorrelation)
			m |= s & 1
			mid[i] = (m + s) / 2
			side[i] = (m - s) / 2
		}
	}
}

// Decorrelate performs inter-channel decorrelation between the samples of the subframes.
// An encoder decorrelates audio samples as follows:
//
//	mid = (left + right)/2
//	side = left - right
func (frame *Frame) Decorrelate() {
	switch frame.Channels {
	case ChannelsLeftSide:
		// 2 channels: left, side; using inter-channel decorrelation
		left := frame.Subframes[0].Samples  // already left; no change after inter-channel decorrelation
		right := frame.Subframes[1].Samples // set to side after inter-channel decorrelation
		for i := range left {
			l := left[i]
			r := right[i]
			// inter-channel decorrelation:
			//	side = left - right
			side := l - r
			right[i] = side
		}
	case ChannelsSideRight:
		// 2 channels: side, right; using inter-channel decorrelation
		left := frame.Subframes[0].Samples  // set to side after inter-channel decorrelation
		right := frame.Subframes[1].Samples // already right; no change after inter-channel decorrelation
		for i := range left {
			l := left[i]
			r := right[i]
			// inter-channel decorrelation:
			//	side = left - right
			side := l - r
			left[i] = side
		}
	case ChannelsMidSide:
		// 2 channels: mid, side; using inter-channel decorrelation
		left := frame.Subframes[0].Samples  // set to mid after inter-channel decorrelation
		right := frame.Subframes[1].Samples // set to side after inter-channel decorrelation
		for i := range left {
			// inter-channel decorrelation:
			//	mid = (left + right)/2
			//	side = left - right
			l := left[i]
			r := right[i]
			mid := int32((int64(l) + int64(r)) >> 1) // NOTE: using `(left + right) >> 1`, not the same as `(left + right) / 2`
			side := l - r
			left[i] = mid
			right[i] = side
		}
	}
}

// unexpected returns io.ErrUnexpectedEOF if error is io.EOF,
// and returns error otherwise.
func unexpected(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}
