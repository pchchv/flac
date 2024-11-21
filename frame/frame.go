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
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"

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

// Parse reads and parses the audio samples from each subframe of the frame.
// If the samples are inter-channel decorrelated between the subframes,
// it correlates them.
func (frame *Frame) Parse() error {
	var err error
	frame.Subframes = make([]*Subframe, frame.Channels.Count())
	for channel := range frame.Subframes {
		// side channel requires an extra bit per sample when
		// using inter-channel decorrelation.
		bps := uint(frame.BitsPerSample)
		switch frame.Channels {
		case ChannelsSideRight:
			// channel 0 is the side channel
			if channel == 0 {
				bps++
			}
		case ChannelsLeftSide, ChannelsMidSide:
			// channel 1 is the side channel
			if channel == 1 {
				bps++
			}
		}

		if frame.Subframes[channel], err = frame.parseSubframe(frame.br, bps); err != nil {
			return err
		}
	}

	// inter-channel correlation of subframe samples
	frame.Correlate()

	// 2 bytes: CRC-16 checksum
	var want uint16
	if err = binary.Read(frame.r, binary.BigEndian, &want); err != nil {
		return unexpected(err)
	}

	if got := frame.crc.Sum16(); got != want {
		return fmt.Errorf("frame.Frame.Parse: CRC-16 checksum mismatch; expected 0x%04X, got 0x%04X", want, got)
	}

	return nil
}

// Hash adds the decoded audio samples of the frame to a running MD5 hash.
// It can be used in conjunction with StreamInfo.MD5sum
// to verify the integrity of the decoded audio samples.
// Note: The audio samples of the frame must be decoded before calling Hash.
func (frame *Frame) Hash(md5sum hash.Hash) {
	var buf [3]byte
	// write decoded samples to a running MD5 hash
	bps := frame.BitsPerSample
	for i := 0; i < int(frame.BlockSize); i++ {
		for _, subframe := range frame.Subframes {
			sample := subframe.Samples[i]
			switch {
			case 1 <= bps && bps <= 8:
				buf[0] = uint8(sample)
				md5sum.Write(buf[:1])
			case 9 <= bps && bps <= 16:
				buf[0] = uint8(sample)
				buf[1] = uint8(sample >> 8)
				md5sum.Write(buf[:2])
			case 17 <= bps && bps <= 24:
				buf[0] = uint8(sample)
				buf[1] = uint8(sample >> 8)
				buf[2] = uint8(sample >> 16)
				md5sum.Write(buf[:])
			default:
				log.Printf("frame.Frame.Hash: support for %d-bit sample size not yet implemented", bps)
			}
		}
	}
}

// SampleNumber returns the first sample number contained within the frame.
func (frame *Frame) SampleNumber() uint64 {
	if frame.HasFixedBlockSize {
		return frame.Num * uint64(frame.BlockSize)
	}
	return frame.Num
}

// parseChannels parses the channels of the header.
func (frame *Frame) parseChannels(br *bits.Reader) error {
	// 4 bits: Channels.
	//
	// The 4 bits are used to specify the channels as follows:
	//    0000: (1 channel) mono.
	//    0001: (2 channels) left, right.
	//    0010: (3 channels) left, right, center.
	//    0011: (4 channels) left, right, left surround, right surround.
	//    0100: (5 channels) left, right, center, left surround, right surround.
	//    0101: (6 channels) left, right, center, LFE, left surround, right surround.
	//    0110: (7 channels) left, right, center, LFE, center surround, side left, side right.
	//    0111: (8 channels) left, right, center, LFE, left surround, right surround, side left, side right.
	//    1000: (2 channels) left, side; using inter-channel decorrelation.
	//    1001: (2 channels) side, right; using inter-channel decorrelation.
	//    1010: (2 channels) mid, side; using inter-channel decorrelation.
	//    1011: reserved.
	//    1100: reserved.
	//    1101: reserved.
	//    1111: reserved.
	x, err := br.Read(4)
	if err != nil {
		return unexpected(err)
	} else if x >= 0xB {
		return fmt.Errorf("frame.Frame.parseHeader: reserved channels bit pattern (%04b)", x)
	}

	frame.Channels = Channels(x)
	return nil
}

// parseSampleRate parses the sample rate of the header.
func (frame *Frame) parseSampleRate(br *bits.Reader, sampleRate uint64) error {
	// The 4 bits are used to specify the sample rate as follows:
	//    0000: unknown sample rate; get from StreamInfo.
	//    0001: 88.2 kHz.
	//    0010: 176.4 kHz.
	//    0011: 192 kHz.
	//    0100: 8 kHz.
	//    0101: 16 kHz.
	//    0110: 22.05 kHz.
	//    0111: 24 kHz.
	//    1000: 32 kHz.
	//    1001: 44.1 kHz.
	//    1010: 48 kHz.
	//    1011: 96 kHz.
	//    1100: get 8 bit sample rate (in kHz) from the end of the header.
	//    1101: get 16 bit sample rate (in Hz) from the end of the header.
	//    1110: get 16 bit sample rate (in daHz) from the end of the header.
	//    1111: invalid.
	switch sampleRate {
	case 0x0:
		// 0000: unknown sample rate; get from StreamInfo
	case 0x1:
		// 0001: 88.2 kHz
		frame.SampleRate = 88200
	case 0x2:
		// 0010: 176.4 kHz
		frame.SampleRate = 176400
		log.Printf("frame.Frame.parseHeader: The flac library test cases do not yet include any audio files with sample rate %d. If possible please consider contributing this audio sample to improve the reliability of the test cases.", frame.SampleRate)
	case 0x3:
		// 0011: 192 kHz
		frame.SampleRate = 192000
	case 0x4:
		// 0100: 8 kHz
		frame.SampleRate = 8000
	case 0x5:
		// 0101: 16 kHz
		frame.SampleRate = 16000
	case 0x6:
		// 0110: 22.05 kHz
		frame.SampleRate = 22050
	case 0x7:
		// 0111: 24 kHz
		frame.SampleRate = 24000
		log.Printf("frame.Frame.parseHeader: The flac library test cases do not yet include any audio files with sample rate %d. If possible please consider contributing this audio sample to improve the reliability of the test cases.", frame.SampleRate)
	case 0x8:
		// 1000: 32 kHz
		frame.SampleRate = 32000
	case 0x9:
		// 1001: 44.1 kHz
		frame.SampleRate = 44100
	case 0xA:
		// 1010: 48 kHz
		frame.SampleRate = 48000
	case 0xB:
		// 1011: 96 kHz
		frame.SampleRate = 96000
	case 0xC:
		// 1100: get 8 bit sample rate (in kHz) from the end of the header
		x, err := br.Read(8)
		if err != nil {
			return unexpected(err)
		}
		frame.SampleRate = uint32(x * 1000)
	case 0xD:
		// 1101: get 16 bit sample rate (in Hz) from the end of the header
		x, err := br.Read(16)
		if err != nil {
			return unexpected(err)
		}
		frame.SampleRate = uint32(x)
	case 0xE:
		// 1110: get 16 bit sample rate (in daHz) from the end of the header
		x, err := br.Read(16)
		if err != nil {
			return unexpected(err)
		}
		frame.SampleRate = uint32(x * 10)
	default:
		// 1111: invalid
		return errors.New("frame.Frame.parseHeader: invalid sample rate bit pattern (1111)")
	}
	return nil
}

// unexpected returns io.ErrUnexpectedEOF if error is io.EOF,
// and returns error otherwise.
func unexpected(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}
