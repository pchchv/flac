package meta

import "strings"

// CueSheetTrackIndex specifies a position within a track.
type CueSheetTrackIndex struct {
	// Index point offset in samples, relative to the track offset.
	Offset uint64
	// Index point number;
	// subsequently incrementing by 1 and always unique within a track.
	Num uint8
}

// CueSheetTrack contains the start offset of
// a track and other track specific metadata.
type CueSheetTrack struct {
	// Track offset in samples,
	// relative to the beginning of the FLAC audio stream.
	Offset uint64
	// Track number; never 0, always unique.
	Num uint8
	// International Standard Recording Code;
	// empty string if not present.
	ISRC string
	// Specifies if the track contains audio or data.
	IsAudio bool
	// Specifies if the track has been recorded with pre-emphasis
	HasPreEmphasis bool
	// Every track has one or more track index points,
	// except for the lead-out track which has zero.
	// Each index point specifies a position within the track.
	Indicies []CueSheetTrackIndex
}

// CueSheet describes how tracks are laid out within a FLAC stream.
type CueSheet struct {
	// Media catalog number.
	MCN string
	// Number of lead-in samples.
	// This field only has meaning for CD-DA cue sheets;
	// for other uses it should be 0.
	// Refer to the spec for additional information.
	NLeadInSamples uint64
	// Specifies if the cue sheet corresponds to a Compact Disc.
	IsCompactDisc bool
	// One or more tracks.
	// The last track of a cue sheet is always the lead-out track.
	Tracks []CueSheetTrack
}

// stringFromSZ returns a copy of the given string terminated at
// the first occurrence of a NULL character.
func stringFromSZ(szStr string) string {
	pos := strings.IndexByte(szStr, '\x00')
	if pos == -1 {
		return szStr
	}
	return string(szStr[:pos])
}
