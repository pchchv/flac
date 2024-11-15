package meta

import (
	"encoding/binary"
	"io"
)

// Application contains third party application specific data.
type Application struct {
	ID   uint32 // registered application ID
	Data []byte
}

// parseApplication reads and parses the body of an Application metadata block.
func (block *Block) parseApplication() (err error) {
	// 32 bits: ID.
	app := new(Application)
	block.Body = app
	if err = binary.Read(block.lr, binary.BigEndian, &app.ID); err != nil {
		return unexpected(err)
	}

	// Check if the Application block only contains an ID.
	if block.Length == 4 {
		return nil
	}

	// (block length)-4 bytes: Data.
	app.Data, err = io.ReadAll(block.lr)
	return unexpected(err)
}
