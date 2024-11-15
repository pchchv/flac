package meta

// Application contains third party application specific data.
type Application struct {
	ID   uint32 // registered application ID
	Data []byte
}
