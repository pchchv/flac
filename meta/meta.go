package meta

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
