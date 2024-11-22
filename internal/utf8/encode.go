// Package utf8 implements encoding and decoding of UTF-8 coded numbers.
package utf8

const (
	tx = 0x80 // 1000 0000
	t2 = 0xC0 // 1100 0000
	t3 = 0xE0 // 1110 0000
	t4 = 0xF0 // 1111 0000
	t5 = 0xF8 // 1111 1000
	t6 = 0xFC // 1111 1100

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
