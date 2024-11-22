// Package utf8 implements encoding and decoding of UTF-8 coded numbers.
package utf8

import (
	"io"

	"github.com/pchchv/flac/internal/ioutilx"
)

const (
	tx = 0x80 // 1000 0000
	t2 = 0xC0 // 1100 0000
	t3 = 0xE0 // 1110 0000
	t4 = 0xF0 // 1111 0000
	t5 = 0xF8 // 1111 1000
	t6 = 0xFC // 1111 1100
	// t7 = 0xFE // 1111 1110
	// t8 = 0xFF // 1111 1111

	maskx = 0x3F // 0011 1111
	mask2 = 0x1F // 0001 1111
	mask3 = 0x0F // 0000 1111
	mask4 = 0x07 // 0000 0111
	mask5 = 0x03 // 0000 0011
	mask6 = 0x01 // 0000 0001

	rune1Max = 1<<7 - 1
	rune2Max = 1<<11 - 1
	rune3Max = 1<<16 - 1
	rune4Max = 1<<21 - 1
	rune5Max = 1<<26 - 1
	rune6Max = 1<<31 - 1
	rune7Max = 1<<36 - 1
)

// Encode encodes x as a "UTF-8" coded number.
func Encode(w io.Writer, x uint64) error {
	// 1-byte, 7-bit sequence
	if x <= rune1Max {
		if err := ioutilx.WriteByte(w, byte(x)); err != nil {
			return err
		}
		return nil
	}

	// get number of continuation bytes and store bits of c0
	var l int       // number of continuation bytes
	var bits uint64 // bits of c0
	switch {
	case x <= rune2Max:
		// if c0 == 110xxxxx
		// total: 11 bits (5 + 6)
		l = 1
		bits = t2 | (x>>6)&mask2
	case x <= rune3Max:
		// if c0 == 1110xxxx
		// total: 16 bits (4 + 6 + 6)
		l = 2
		bits = t3 | (x>>(6*2))&mask3
	case x <= rune4Max:
		// if c0 == 11110xxx
		// total: 21 bits (3 + 6 + 6 + 6)
		l = 3
		bits = t4 | (x>>(6*3))&mask4
	case x <= rune5Max:
		// if c0 == 111110xx
		// total: 26 bits (2 + 6 + 6 + 6 + 6)
		l = 4
		bits = t5 | (x>>(6*4))&mask5
	case x <= rune6Max:
		// if c0 == 1111110x
		// total: 31 bits (1 + 6 + 6 + 6 + 6 + 6)
		l = 5
		bits = t6 | (x>>(6*5))&mask6
	case x <= rune7Max:
		// if c0 == 11111110
		// total: 36 bits (0 + 6 + 6 + 6 + 6 + 6 + 6)
		l = 6
		bits = 0
	}

	// store bits of c0
	if err := ioutilx.WriteByte(w, byte(bits)); err != nil {
		return err
	}

	// store continuation bytes
	for i := l - 1; i >= 0; i-- {
		bits := tx | (x>>uint(6*i))&maskx
		if err := ioutilx.WriteByte(w, byte(bits)); err != nil {
			return err
		}
	}
	return nil
}
