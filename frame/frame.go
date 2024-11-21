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

// unexpected returns io.ErrUnexpectedEOF if error is io.EOF,
// and returns error otherwise.
func unexpected(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}
