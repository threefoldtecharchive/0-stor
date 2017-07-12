package db

// Model is the interface your object has to implement to be usable by the DB object
type Model interface {
	Key() string // Get key of the object used in Get()
	Decode([]byte) error
	Encode() ([]byte, error)
	// Get(item *Model) error
	// Save(item *Model) error
	// Delete(item *Model)
	// Exists(item *Model) (bool, error)
	// New() *Model
	Validate() error
}
