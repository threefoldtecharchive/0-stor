package api

// GRPC metadata keys used for 0-stor
const (
	// GRPCMetaAuthKey represents the key
	// where the JWT can be found in the GRPC's metadata
	GRPCMetaAuthKey = "authorization"

	// GRPCMetaLabelKey represents the full namespace label,
	// which can be found in the GRPC's metadata
	GRPCMetaLabelKey = "label"
)
