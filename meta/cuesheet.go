package meta

// A CueSheetTrackIndex specifies a position within a track.
type CueSheetTrackIndex struct {
	// Index point offset in samples, relative to the track offset.
	Offset uint64
	// Index point number;
	// subsequently incrementing by 1 and always unique within a track.
	Num uint8
}
