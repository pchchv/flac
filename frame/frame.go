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
