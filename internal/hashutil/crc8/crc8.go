package crc8

// Size of a CRC-8 checksum in bytes.
const Size = 1

// digest represents the partial evaluation of a checksum.
type digest struct {
	crc   uint8
	table *Table
}

func (d *digest) Size() int {
	return Size
}

func (d *digest) BlockSize() int {
	return 1
}

func (d *digest) Reset() {
	d.crc = 0
}

// Table is a 256-word table representing
// the polynomial for efficient processing.
type Table [256]uint8
