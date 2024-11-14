package meta

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// VorbisComment contains a list of name-value pairs.
type VorbisComment struct {
	Vendor string      // vendor name
	Tags   [][2]string // // list of tags, each represented by a name-value pair
}

// parseVorbisComment reads and parses the body of a VorbisComment metadata
// block.
func (block *Block) parseVorbisComment() (err error) {
	// 32 bits: vendor length.
	var x uint32
	if err = binary.Read(block.lr, binary.LittleEndian, &x); err != nil {
		return unexpected(err)
	}

	// (vendor length) bits: Vendor.
	vendor, err := readString(block.lr, int(x))
	if err != nil {
		return unexpected(err)
	}

	comment := new(VorbisComment)
	block.Body = comment
	comment.Vendor = vendor

	// Parse tags.
	// 32 bits: number of tags.
	if err = binary.Read(block.lr, binary.LittleEndian, &x); err != nil {
		return unexpected(err)
	}

	if x < 1 {
		return nil
	}

	comment.Tags = make([][2]string, x)
	for i := range comment.Tags {
		// 32 bits: vector length
		if err = binary.Read(block.lr, binary.LittleEndian, &x); err != nil {
			return unexpected(err)
		}

		// (vector length): vector.
		vector, err := readString(block.lr, int(x))
		if err != nil {
			return unexpected(err)
		}

		// Parse tag, which has the following format:
		//    NAME=VALUE
		pos := strings.Index(vector, "=")
		if pos == -1 {
			return fmt.Errorf("meta.Block.parseVorbisComment: unable to locate '=' in vector %q", vector)
		}
		comment.Tags[i][0] = vector[:pos]
		comment.Tags[i][1] = vector[pos+1:]
	}

	return nil
}
