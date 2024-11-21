package crc8

// digest represents the partial evaluation of a checksum.
type digest struct {
	crc   uint8
	table *Table
}

// Table is a 256-word table representing
// the polynomial for efficient processing.
type Table [256]uint8
