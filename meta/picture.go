package meta

import (
	"encoding/binary"
	"io"
)

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

// parsePicture reads and parses the body of a Picture metadata block.
func (block *Block) parsePicture() (err error) {
	// 32 bits: Type.
	pic := new(Picture)
	block.Body = pic
	if err = binary.Read(block.lr, binary.BigEndian, &pic.Type); err != nil {
		return unexpected(err)
	}

	// 32 bits: (MIME type length).
	var x uint32
	if err = binary.Read(block.lr, binary.BigEndian, &x); err != nil {
		return unexpected(err)
	}

	// (MIME type length) bytes: MIME.
	if pic.MIME, err = readString(block.lr, int(x)); err != nil {
		return unexpected(err)
	}

	// 32 bits: (description length).
	if err = binary.Read(block.lr, binary.BigEndian, &x); err != nil {
		return unexpected(err)
	}

	// (description length) bytes: Desc.
	if pic.Desc, err = readString(block.lr, int(x)); err != nil {
		return unexpected(err)
	}

	// 32 bits: Width.
	if err = binary.Read(block.lr, binary.BigEndian, &pic.Width); err != nil {
		return unexpected(err)
	}

	// 32 bits: Height.
	if err = binary.Read(block.lr, binary.BigEndian, &pic.Height); err != nil {
		return unexpected(err)
	}

	// 32 bits: Depth.
	if err = binary.Read(block.lr, binary.BigEndian, &pic.Depth); err != nil {
		return unexpected(err)
	}

	// 32 bits: NPalColors.
	if err = binary.Read(block.lr, binary.BigEndian, &pic.NPalColors); err != nil {
		return unexpected(err)
	}

	// 32 bits: (data length).
	if err = binary.Read(block.lr, binary.BigEndian, &x); err != nil {
		return unexpected(err)
	}

	if x == 0 {
		return nil
	}

	// (data length) bytes: Data.
	pic.Data = make([]byte, x)
	_, err = io.ReadFull(block.lr, pic.Data)
	return unexpected(err)
}
