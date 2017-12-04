package grpc

// GRPC metadata keys used for 0-stor
const (
	// MetaAuthKey represents the key where the JWT can be found in the GRPC's metadata
	MetaAuthKey = "authorization"

	// MetaLabelKey  represents the full namespace label can be found in the GRPC's metadata
	MetaLabelKey = "label"
)
