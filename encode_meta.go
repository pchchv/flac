package flac

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/icza/bitio"
	"github.com/pchchv/flac/internal/ioutilx"
	"github.com/pchchv/flac/meta"
)

// encodeBlockHeader encodes the metadata block header, writing to bw.
func encodeBlockHeader(bw *bitio.Writer, hdr *meta.Header) error {
	// 1 bit: IsLast
	if err := bw.WriteBool(hdr.IsLast); err != nil {
		return err
	}

	// 7 bits: Type
	if err := bw.WriteBits(uint64(hdr.Type), 7); err != nil {
		return err
	}

	// 24 bits: Length
	if err := bw.WriteBits(uint64(hdr.Length), 24); err != nil {
		return err
	}

	return nil
}

// encodePadding encodes the Padding metadata block, writing to bw.
func encodePadding(bw *bitio.Writer, length int64, last bool) error {
	// store metadata block header
	hdr := &meta.Header{
		IsLast: last,
		Type:   meta.TypePadding,
		Length: length,
	}

	if err := encodeBlockHeader(bw, hdr); err != nil {
		return err
	}

	// store metadata block body
	if _, err := io.CopyN(bw, ioutilx.Zero, length); err != nil {
		return err
	}

	return nil
}

// encodeEmptyBlock encodes the metadata block header of an
// empty metadata block with the specified type,
// writing to bw.
func encodeEmptyBlock(bw *bitio.Writer, typ meta.Type, last bool) error {
	// store metadata block header
	hdr := &meta.Header{
		IsLast: last,
		Type:   typ,
		Length: 0,
	}
	if err := encodeBlockHeader(bw, hdr); err != nil {
		return err
	}
	return nil
}

// encodeStreamInfo encodes the StreamInfo metadata block, writing to bw.
func encodeStreamInfo(bw *bitio.Writer, info *meta.StreamInfo, last bool) error {
	// store metadata block header
	const nbits = 16 + 16 + 24 + 24 + 20 + 3 + 5 + 36 + 8*16
	hdr := &meta.Header{
		IsLast: last,
		Type:   meta.TypeStreamInfo,
		Length: nbits / 8,
	}
	if err := encodeBlockHeader(bw, hdr); err != nil {
		return err
	}

	// store metadata block body
	// 16 bits: BlockSizeMin
	if err := bw.WriteBits(uint64(info.BlockSizeMin), 16); err != nil {
		return err
	}

	// 16 bits: BlockSizeMax
	if err := bw.WriteBits(uint64(info.BlockSizeMax), 16); err != nil {
		return err
	}

	// 24 bits: FrameSizeMin
	if err := bw.WriteBits(uint64(info.FrameSizeMin), 24); err != nil {
		return err
	}

	// 24 bits: FrameSizeMax
	if err := bw.WriteBits(uint64(info.FrameSizeMax), 24); err != nil {
		return err
	}

	// 20 bits: SampleRate
	if err := bw.WriteBits(uint64(info.SampleRate), 20); err != nil {
		return err
	}

	// 3 bits: NChannels; stored as (number of channels) - 1
	if err := bw.WriteBits(uint64(info.NChannels-1), 3); err != nil {
		return err
	}

	// 5 bits: BitsPerSample; stored as (bits-per-sample) - 1
	if err := bw.WriteBits(uint64(info.BitsPerSample-1), 5); err != nil {
		return err
	}

	// 36 bits: NSamples
	if err := bw.WriteBits(info.NSamples, 36); err != nil {
		return err
	}

	// 16 bytes: MD5sum
	if _, err := bw.Write(info.MD5sum[:]); err != nil {
		return err
	}

	return nil
}

// encodeApplication encodes the Application metadata block, writing to bw.
func encodeApplication(bw *bitio.Writer, app *meta.Application, last bool) error {
	// store metadata block header
	nbits := int64(32 + 8*len(app.Data))
	hdr := &meta.Header{
		IsLast: last,
		Type:   meta.TypeApplication,
		Length: nbits / 8,
	}
	if err := encodeBlockHeader(bw, hdr); err != nil {
		return err
	}

	// store metadata block body
	// 32 bits: ID
	if err := bw.WriteBits(uint64(app.ID), 32); err != nil {
		return err
	}

	if _, err := bw.Write(app.Data); err != nil {
		return err
	}

	return nil
}

// encodeSeekTable encodes the SeekTable metadata block, writing to bw.
func encodeSeekTable(bw *bitio.Writer, table *meta.SeekTable, last bool) error {
	// store metadata block header
	nbits := int64((64 + 64 + 16) * len(table.Points))
	hdr := &meta.Header{
		IsLast: last,
		Type:   meta.TypeSeekTable,
		Length: nbits / 8,
	}
	if err := encodeBlockHeader(bw, hdr); err != nil {
		return err
	}

	// store metadata block body
	for _, point := range table.Points {
		if err := binary.Write(bw, binary.BigEndian, point); err != nil {
			return err
		}
	}

	return nil
}

// encodeVorbisComment encodes the VorbisComment metadata block, writing to bw.
func encodeVorbisComment(bw *bitio.Writer, comment *meta.VorbisComment, last bool) error {
	// store metadata block header
	nbits := int64(32 + 8*len(comment.Vendor) + 32)
	for _, tag := range comment.Tags {
		nbits += int64(32 + 8*(len(tag[0])+1+len(tag[1])))
	}
	hdr := &meta.Header{
		IsLast: last,
		Type:   meta.TypeVorbisComment,
		Length: nbits / 8,
	}
	if err := encodeBlockHeader(bw, hdr); err != nil {
		return err
	}

	// store metadata block body
	// 32 bits: vendor length
	if err := binary.Write(bw, binary.LittleEndian, uint32(len(comment.Vendor))); err != nil {
		return err
	}

	// (vendor length) bits: Vendor
	if _, err := bw.Write([]byte(comment.Vendor)); err != nil {
		return err
	}

	// store tags
	// 32 bits: number of tags
	if err := binary.Write(bw, binary.LittleEndian, uint32(len(comment.Tags))); err != nil {
		return err
	}

	for _, tag := range comment.Tags {
		// store tag, which has the following format: NAME=VALUE
		buf := []byte(fmt.Sprintf("%s=%s", tag[0], tag[1]))
		// 32 bits: vector length
		if err := binary.Write(bw, binary.LittleEndian, uint32(len(buf))); err != nil {
			return err
		}

		// (vector length): vector
		if _, err := bw.Write(buf); err != nil {
			return err
		}
	}

	return nil
}

// encodeCueSheet encodes the CueSheet metadata block, writing to bw.
func encodeCueSheet(bw *bitio.Writer, cs *meta.CueSheet, last bool) error {
	// store metadata block header
	nbits := int64(8*128 + 64 + 1 + 7 + 8*258 + 8)
	for _, track := range cs.Tracks {
		nbits += 64 + 8 + 8*12 + 1 + 1 + 6 + 8*13 + 8
		for range track.Indicies {
			nbits += 64 + 8 + 8*3
		}
	}
	hdr := &meta.Header{
		IsLast: last,
		Type:   meta.TypeCueSheet,
		Length: nbits / 8,
	}
	if err := encodeBlockHeader(bw, hdr); err != nil {
		return err
	}

	// store metadata block body
	// store cue sheet
	// 128 bytes: MCN
	var mcn [128]byte
	copy(mcn[:], cs.MCN)
	if _, err := bw.Write(mcn[:]); err != nil {
		return err
	}

	// 64 bits: NLeadInSamples
	if err := bw.WriteBits(cs.NLeadInSamples, 64); err != nil {
		return err
	}

	// 1 bit: IsCompactDisc
	if err := bw.WriteBool(cs.IsCompactDisc); err != nil {
		return err
	}

	// 7 bits and 258 bytes: reserved
	if err := bw.WriteBits(0, 7); err != nil {
		return err
	}

	if _, err := io.CopyN(bw, ioutilx.Zero, 258); err != nil {
		return err
	}

	// store cue sheet tracks
	// 8 bits: (number of tracks)
	if err := bw.WriteBits(uint64(len(cs.Tracks)), 8); err != nil {
		return err
	}

	for _, track := range cs.Tracks {
		// 64 bits: Offset
		if err := bw.WriteBits(track.Offset, 64); err != nil {
			return err
		}

		// 8 bits: Num
		if err := bw.WriteBits(uint64(track.Num), 8); err != nil {
			return err
		}

		// 12 bytes: ISRC
		var isrc [12]byte
		copy(isrc[:], track.ISRC)
		if _, err := bw.Write(isrc[:]); err != nil {
			return err
		}

		// 1 bit: IsAudio
		if err := bw.WriteBool(!track.IsAudio); err != nil {
			return err
		}

		// 1 bit: HasPreEmphasis
		// mask = 01000000
		if err := bw.WriteBool(track.HasPreEmphasis); err != nil {
			return err
		}

		// 6 bits and 13 bytes: reserved
		// mask = 00111111
		if err := bw.WriteBits(0, 6); err != nil {
			return err
		}

		if _, err := io.CopyN(bw, ioutilx.Zero, 13); err != nil {
			return err
		}

		// Store indicies
		// 8 bits: (number of indicies)
		if err := bw.WriteBits(uint64(len(track.Indicies)), 8); err != nil {
			return err
		}

		for _, index := range track.Indicies {
			// 64 bits: Offset
			if err := bw.WriteBits(index.Offset, 64); err != nil {
				return err
			}

			// 8 bits: Num
			if err := bw.WriteBits(uint64(index.Num), 8); err != nil {
				return err
			}

			// 3 bytes: reserved
			if _, err := io.CopyN(bw, ioutilx.Zero, 3); err != nil {
				return err
			}
		}
	}

	return nil
}
