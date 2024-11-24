package flac

import (
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
