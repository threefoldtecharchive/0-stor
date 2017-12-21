package server

type (
	// Object is the data structure for all data (blobs) stored in the 0-stor server.
	// It is used to exchange it between the codebase.
	// The decoding and encoding happens in the encoding package.
	Object struct {
		// Data in its raw encoded form
		Data []byte
	}

	// Namespace is the data structure for all namespaces referenced and stored in the 0-stor server.
	// It is used to exchange namespaces between the codebase.
	// The decoding and encoding happens in the encoding package.
	Namespace struct {
		// Reserved (total) space in this namespace (in bytes)
		Reserved uint64
		// Label (or name) of the namespace
		Label []byte
	}

	// StoreStat is the data structure for the global store statistics,
	// stored for a single 0-stor server. One per server and thus one per database.
	// It is used to exchange the stats between the codebase.
	// The decoding and encoding happens in the encoding package.
	StoreStat struct {
		// Available disk space in bytes
		SizeAvailable uint64
		// Space used in bytes
		SizeUsed uint64
	}
)

// ObjectStatus represents the status received after checking,
// whether or not an object is ok
type ObjectStatus uint8

const (
	// ObjectStatusMissing indicates the requested object doesn't exist.
	ObjectStatusMissing ObjectStatus = iota
	// ObjectStatusOK indicates the requested object exists and is healthy
	ObjectStatusOK
	// ObjectStatusCorrupted indicates the requested object exists,
	// but its checksum indicates that the stored data is corrupted.
	ObjectStatusCorrupted
)

// String implements Stringer.String
func (status ObjectStatus) String() string {
	str, ok := _ObjectStatusValueStringMapping[status]
	if !ok {
		return ""
	}
	return str
}

const _ObjectStatusStrings = "okcorruptedmissing"

var _ObjectStatusValueStringMapping = map[ObjectStatus]string{
	ObjectStatusOK:        _ObjectStatusStrings[:2],
	ObjectStatusCorrupted: _ObjectStatusStrings[2:11],
	ObjectStatusMissing:   _ObjectStatusStrings[11:],
}
