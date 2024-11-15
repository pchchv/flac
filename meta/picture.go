package meta

// Picture contains the image data of an embedded picture.
type Picture struct {
	// Picture type according to the ID3v2 APIC frame:
	//     0: Other
	//     1: 32x32 pixels 'file icon' (PNG only)
	//     2: Other file icon
	//     3: Cover (front)
	//     4: Cover (back)
	//     5: Leaflet page
	//     6: Media (e.g. label side of CD)
	//     7: Lead artist/lead performer/soloist
	//     8: Artist/performer
	//     9: Conductor
	//    10: Band/Orchestra
	//    11: Composer
	//    12: Lyricist/text writer
	//    13: Recording Location
	//    14: During recording
	//    15: During performance
	//    16: Movie/video screen capture
	//    17: A bright coloured fish
	//    18: Illustration
	//    19: Band/artist logotype
	//    20: Publisher/Studio logotype
	Type uint32
	// MIME type string.
	// The MIME type "-->" specifies that the picture data is
	// to be interpreted as an URL instead of image data.
	MIME string
	// Description of the picture.
	Desc string
	// Image dimensions.
	Width, Height uint32
	// Color depth in bits-per-pixel.
	Depth uint32
	// Number of colors in palette;
	// 0 for non-indexed images.
	NPalColors uint32
	// Image data.
	Data []byte
}
