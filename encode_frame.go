package flac

import (
	"fmt"
	"math"

	"github.com/icza/bitio"
	"github.com/pchchv/flac/frame"
)

// encodeFrameHeaderBitsPerSample encodes the bits-per-sample of the frame header,
// writing to bw.
func encodeFrameHeaderBitsPerSample(bw *bitio.Writer, bps uint8) error {
	// sample size in bits:
	//    000 : get from STREAMINFO metadata block
	//    001 : 8 bits per sample
	//    010 : 12 bits per sample
	//    011 : reserved
	//    100 : 16 bits per sample
	//    101 : 20 bits per sample
	//    110 : 24 bits per sample
	//    111 : reserved
	var bits uint64
	switch bps {
	case 0:
		// 000 : get from STREAMINFO metadata block
		bits = 0x0
	case 8:
		// 001 : 8 bits per sample
		bits = 0x1
	case 12:
		// 010 : 12 bits per sample
		bits = 0x2
	case 16:
		// 100 : 16 bits per sample
		bits = 0x4
	case 20:
		// 101 : 20 bits per sample
		bits = 0x5
	case 24:
		// 110 : 24 bits per sample
		bits = 0x6
	default:
		return fmt.Errorf("support for sample size %v not yet implemented", bps)
	}

	if err := bw.WriteBits(bits, 3); err != nil {
		return err
	}

	return nil
}

// encodeFrameHeaderBlockSize encodes the block size of the frame header,
// writing to bw.
// It returns the number of bits used to store block size after the frame header.
func encodeFrameHeaderBlockSize(bw *bitio.Writer, blockSize uint16) (nblockSizeSuffixBits byte, err error) {
	// block size in inter-channel samples:
	//    0000 : reserved
	//    0001 : 192 samples
	//    0010-0101 : 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608
	//    0110 : get 8 bit (blocksize-1) from end of header
	//    0111 : get 16 bit (blocksize-1) from end of header
	//    1000-1111 : 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/16384/32768
	var bits uint64
	switch blockSize {
	case 192:
		// 0001
		bits = 0x1
	case 576, 1152, 2304, 4608:
		// 0010-0101 : 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608
		bits = 0x2 + uint64(math.Log2(float64(blockSize/576)))
	case 256, 512, 1024, 2048, 4096, 8192, 16384, 32768:
		// 1000-1111 : 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/16384/32768
		bits = 0x8 + uint64(math.Log2(float64(blockSize/256)))
	default:
		if blockSize <= 256 {
			// 0110 : get 8 bit (blocksize-1) from end of header
			bits = 0x6
			nblockSizeSuffixBits = 8
		} else {
			// 0111 : get 16 bit (blocksize-1) from end of header
			bits = 0x7
			nblockSizeSuffixBits = 16
		}
	}

	if err := bw.WriteBits(bits, 4); err != nil {
		return 0, err
	}

	return nblockSizeSuffixBits, nil
}

// encodeFrameHeaderSampleRate encodes the sample rate of the frame header,
// writing to bw.
// It returns the bits and the number of bits used to store sample rate after the frame header.
func encodeFrameHeaderSampleRate(bw *bitio.Writer, sampleRate uint32) (sampleRateSuffixBits uint64, nsampleRateSuffixBits byte, err error) {
	// sample rate:
	//    0000 : get from STREAMINFO metadata block
	//    0001 : 88.2kHz
	//    0010 : 176.4kHz
	//    0011 : 192kHz
	//    0100 : 8kHz
	//    0101 : 16kHz
	//    0110 : 22.05kHz
	//    0111 : 24kHz
	//    1000 : 32kHz
	//    1001 : 44.1kHz
	//    1010 : 48kHz
	//    1011 : 96kHz
	//    1100 : get 8 bit sample rate (in kHz) from end of header
	//    1101 : get 16 bit sample rate (in Hz) from end of header
	//    1110 : get 16 bit sample rate (in tens of Hz) from end of header
	//    1111 : invalid, to prevent sync-fooling string of 1s
	var bits uint64
	switch sampleRate {
	case 0:
		// 0000 : get from STREAMINFO metadata block
		bits = 0
	case 88200:
		// 0001 : 88.2kHz
		bits = 0x1
	case 176400:
		// 0010 : 176.4kHz
		bits = 0x2
	case 192000:
		// 0011 : 192kHz
		bits = 0x3
	case 8000:
		// 0100 : 8kHz
		bits = 0x4
	case 16000:
		// 0101 : 16kHz
		bits = 0x5
	case 22050:
		// 0110 : 22.05kHz
		bits = 0x6
	case 24000:
		// 0111 : 24kHz
		bits = 0x7
	case 32000:
		// 1000 : 32kHz
		bits = 0x8
	case 44100:
		// 1001 : 44.1kHz
		bits = 0x9
	case 48000:
		// 1010 : 48kHz
		bits = 0xA
	case 96000:
		// 1011 : 96kHz
		bits = 0xB
	default:
		switch {
		case sampleRate <= 255000 && sampleRate%1000 == 0:
			// 1100 : get 8 bit sample rate (in kHz) from end of header
			bits = 0xC
			sampleRateSuffixBits = uint64(sampleRate / 1000)
			nsampleRateSuffixBits = 8
		case sampleRate <= 65535:
			// 1101 : get 16 bit sample rate (in Hz) from end of header
			bits = 0xD
			sampleRateSuffixBits = uint64(sampleRate)
			nsampleRateSuffixBits = 16
		case sampleRate <= 655350 && sampleRate%10 == 0:
			// 1110 : get 16 bit sample rate (in tens of Hz) from end of header
			bits = 0xE
			sampleRateSuffixBits = uint64(sampleRate / 10)
			nsampleRateSuffixBits = 16
		default:
			return 0, 0, fmt.Errorf("unable to encode sample rate %v", sampleRate)
		}
	}

	if err := bw.WriteBits(bits, 4); err != nil {
		return 0, 0, err
	}

	return sampleRateSuffixBits, nsampleRateSuffixBits, nil
}

// encodeFrameHeaderChannels encodes the channels assignment of the frame header,
// writing to bw.
func encodeFrameHeaderChannels(bw *bitio.Writer, channels frame.Channels) error {
	// channel assignment.
	//    0000-0111 : (number of independent channels)-1. Where defined, the channel order follows SMPTE/ITU-R recommendations. The assignments are as follows:
	//        1 channel: mono
	//        2 channels: left, right
	//        3 channels: left, right, center
	//        4 channels: front left, front right, back left, back right
	//        5 channels: front left, front right, front center, back/surround left, back/surround right
	//        6 channels: front left, front right, front center, LFE, back/surround left, back/surround right
	//        7 channels: front left, front right, front center, LFE, back center, side left, side right
	//        8 channels: front left, front right, front center, LFE, back left, back right, side left, side right
	//    1000 : left/side stereo: channel 0 is the left channel, channel 1 is the side(difference) channel
	//    1001 : right/side stereo: channel 0 is the side(difference) channel, channel 1 is the right channel
	//    1010 : mid/side stereo: channel 0 is the mid(average) channel, channel 1 is the side(difference) channel
	//    1011-1111 : reserved
	var bits uint64
	switch channels {
	case frame.ChannelsMono, frame.ChannelsLR, frame.ChannelsLRC, frame.ChannelsLRLsRs, frame.ChannelsLRCLsRs, frame.ChannelsLRCLfeLsRs, frame.ChannelsLRCLfeCsSlSr, frame.ChannelsLRCLfeLsRsSlSr:
		// 1 channel: mono
		// 2 channels: left, right
		// 3 channels: left, right, center
		// 4 channels: left, right, left surround, right surround
		// 5 channels: left, right, center, left surround, right surround
		// 6 channels: left, right, center, LFE, left surround, right surround
		// 7 channels: left, right, center, LFE, center surround, side left, side right
		// 8 channels: left, right, center, LFE, left surround, right surround, side left, side right
		bits = uint64(channels.Count() - 1)
	case frame.ChannelsLeftSide:
		// 2 channels: left, side; using inter-channel decorrelation
		// 1000 : left/side stereo: channel 0 is the left channel, channel 1 is the side(difference) channel
		bits = 0x8
	case frame.ChannelsSideRight:
		// 2 channels: side, right; using inter-channel decorrelation
		// 1001 : right/side stereo: channel 0 is the side(difference) channel, channel 1 is the right channel
		bits = 0x9
	case frame.ChannelsMidSide:
		// 2 channels: mid, side; using inter-channel decorrelation
		// 1010 : mid/side stereo: channel 0 is the mid(average) channel, channel 1 is the side(difference) channel
		bits = 0xA
	default:
		return fmt.Errorf("support for channel assignment %v not yet implemented", channels)
	}

	if err := bw.WriteBits(bits, 4); err != nil {
		return err
	}

	return nil
}
