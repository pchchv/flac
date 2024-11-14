// Package meta package implements access to FLAC metadata blocks.
//
// The following is a brief introduction to the FLAC metadata format.
// FLAC metadata is stored in blocks; each block contains a header followed by a body.
// The block header describes the body type of the block, its length in bytes,
// and specifies whether the block was the last metadata block in the FLAC stream.
// The contents of the block body depend on the type specified in the block header.
//
// As of this writing, the FLAC metadata format defines seven different types of metadata blocks
// (StreamInfo, Padding, Application, SeekTable, VorbisComment, CueSheet, Picture).
package meta

import "io"

// Metadata block body types.
const (
	TypeStreamInfo    Type = 0
	TypePadding       Type = 1
	TypeApplication   Type = 2
	TypeSeekTable     Type = 3
	TypeVorbisComment Type = 4
	TypeCueSheet      Type = 5
	TypePicture       Type = 6
)

// Type represents the type of a metadata block body.
type Type uint8

func (t Type) String() string {
	switch t {
	case TypeStreamInfo:
		return "stream info"
	case TypePadding:
		return "padding"
	case TypeApplication:
		return "application"
	case TypeSeekTable:
		return "seek table"
	case TypeVorbisComment:
		return "vorbis comment"
	case TypeCueSheet:
		return "cue sheet"
	case TypePicture:
		return "picture"
	default:
		return "<unknown block type>"
	}
}

// Header contains information about the
// type and length of a metadata block.
type Header struct {
	Type   Type  // metadata block body type
	Length int64 // length of body data in bytes
	IsLast bool  // specifies if the block is the last metadata block
}

// Block contains the header and body of a metadata block.
type Block struct {
	// Metadata block header.
	Header
	// Metadata block body of type *StreamInfo, *Application, ... etc.
	// Body is initially nil,
	// and gets populated by a call to Block.Parse.
	Body interface{}
	// Underlying io.Reader; limited by the length of the block body.
	lr io.Reader
}

// unexpected returns io.ErrUnexpectedEOF if err is io.EOF,
// and returns error otherwise.
func unexpected(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}

	return err
}
