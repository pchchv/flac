package flac

import (
	"fmt"
	"math"

	"github.com/icza/bitio"
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
