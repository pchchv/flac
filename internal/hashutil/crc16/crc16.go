package crc16

// Size of a CRC-16 checksum in bytes.
const Size = 2

// Table is a 256-word table representing the
// polynomial for efficient processing.
type Table [256]uint16

// digest represents the partial evaluation of a checksum.
type digest struct {
	crc   uint16
	table *Table
}

// Sum16 returns the 16-bit checksum of the hash.
func (d *digest) Sum16() uint16 {
	return d.crc
}

func (d *digest) Sum(in []byte) []byte {
	s := d.Sum16()
	return append(in, byte(s>>8), byte(s))
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

// Update returns the result of adding the bytes in p to the crc.
func Update(crc uint16, table *Table, p []byte) uint16 {
	for _, v := range p {
		crc = crc<<8 ^ table[crc>>8^uint16(v)]
	}
	return crc
}
