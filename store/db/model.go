package db

// Model is the interface your object has to implement to be usable by the DB object
type Model interface {
	/*
		Keys may be prefixed internally.
		This method is used in Used in Get() & Set()
	 */
	Key() string
	Decode([]byte) error
	Encode() ([]byte, error)
	Validate() error
}
