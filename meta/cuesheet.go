package meta

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

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

// parseTrack parses the i:th cue sheet track,
// and ensures that its track number is unique.
func (block *Block) parseTrack(cs *CueSheet, i int, uniq map[uint8]struct{}) (err error) {
	track := &cs.Tracks[i]
	// 64 bits: Offset.
	if err = binary.Read(block.lr, binary.BigEndian, &track.Offset); err != nil {
		return unexpected(err)
	}

	if cs.IsCompactDisc && track.Offset%588 != 0 {
		return fmt.Errorf("meta.Block.parseCueSheet: CD-DA track offset (%d) must be evenly divisible by 588", track.Offset)
	}

	// 8 bits: Num.
	if err := binary.Read(block.lr, binary.BigEndian, &track.Num); err != nil {
		return unexpected(err)
	}

	if _, ok := uniq[track.Num]; ok {
		return fmt.Errorf("meta.Block.parseCueSheet: duplicated track number %d", track.Num)
	}

	uniq[track.Num] = struct{}{}
	if track.Num == 0 {
		return errors.New("meta.Block.parseCueSheet: invalid track number (0)")
	}

	isLeadOut := i == len(cs.Tracks)-1
	if cs.IsCompactDisc {
		if !isLeadOut && track.Num >= 100 {
			return fmt.Errorf("meta.Block.parseCueSheet: CD-DA track number (%d) exceeds 99", track.Num)
		} else if track.Num != 170 {
			return fmt.Errorf("meta.Block.parseCueSheet: invalid lead-out CD-DA track number; expected 170, got %d", track.Num)
		}
	} else if isLeadOut && track.Num != 255 {
		return fmt.Errorf("meta.Block.parseCueSheet: invalid lead-out track number; expected 255, got %d", track.Num)
	}

	// 12 bytes: ISRC.
	szISRC, err := readString(block.lr, 12)
	if err != nil {
		return unexpected(err)
	}
	track.ISRC = stringFromSZ(szISRC)

	// 1 bit: IsAudio.
	var x uint8
	if err = binary.Read(block.lr, binary.BigEndian, &x); err != nil {
		return unexpected(err)
	}

	// mask = 10000000
	if x&0x80 == 0 {
		track.IsAudio = true
	}

	// 1 bit: HasPreEmphasis.
	// mask = 01000000
	if x&0x40 != 0 {
		track.HasPreEmphasis = true
	}

	// 6 bits and 13 bytes: reserved.
	// mask = 00111111
	if x&0x3F != 0 {
		return ErrInvalidPadding
	}

	lr := io.LimitReader(block.lr, 13)
	zr := zeros{r: lr}
	if _, err = io.Copy(io.Discard, zr); err != nil {
		return err
	}

	// parse indicies
	// 8 bits: (number of indicies)
	if err = binary.Read(block.lr, binary.BigEndian, &x); err != nil {
		return unexpected(err)
	}

	if x < 1 {
		if !isLeadOut {
			return errors.New("meta.Block.parseCueSheet: at least one track index required")
		}
		// lead-out track has no track indices to parse; return early.
		return nil
	}

	track.Indicies = make([]CueSheetTrackIndex, x)
	for i := range track.Indicies {
		index := &track.Indicies[i]
		// 64 bits: Offset.
		if err = binary.Read(block.lr, binary.BigEndian, &index.Offset); err != nil {
			return unexpected(err)
		}

		// 8 bits: Num.
		if err = binary.Read(block.lr, binary.BigEndian, &index.Num); err != nil {
			return unexpected(err)
		}

		// 3 bytes: reserved.
		lr = io.LimitReader(block.lr, 3)
		zr = zeros{r: lr}
		if _, err = io.Copy(io.Discard, zr); err != nil {
			return err
		}
	}

	return nil
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
