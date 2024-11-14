package meta

// VorbisComment contains a list of name-value pairs.
type VorbisComment struct {
	Vendor string      // vendor name
	Tags   [][2]string // // list of tags, each represented by a name-value pair
}
