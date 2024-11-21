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

// Sum8 returns the 8-bit checksum of the hash.
func (d *digest) Sum8() uint8 {
	return d.crc
}

func (d *digest) Sum(in []byte) []byte {
	return append(in, d.crc)
}

// Table is a 256-word table representing
// the polynomial for efficient processing.
type Table [256]uint8

// Update returns the result of adding the bytes in p to the crc.
func Update(crc uint8, table *Table, p []byte) uint8 {
	for _, v := range p {
		crc = table[crc^v]
	}
	return crc
}
