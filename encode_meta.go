package flac

import (
	"encoding/binary"
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
