package encoding

import "github.com/zero-os/0-stor/client/meta"

// MarshalMetadata returns the encoding of the data parameter.
// It is important to use this function with a matching `UnmarshalMetadata` function.
type MarshalMetadata func(data meta.Data) ([]byte, error)

// UnmarshalMetadata parses the encoded data
// and stores the result in the value pointed to by the data parameter.
// It is important to use this function with a matching `MashalMetadata` function.
type UnmarshalMetadata func(b []byte, data *meta.Data) error
