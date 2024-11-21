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

// Channels specifies the number of channels (subframes) that exist in a frame,
// their order and possible inter-channel decorrelation.
type Channels uint8
