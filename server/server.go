package server

import (
	"github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/jwt"
)

// New creates a 0-stor server with given DB data & meta directory
func New(data, meta string, jwtVerifier jwt.TokenVerifier, maxSizeMsg int) (api.API, error) {
	db, err := badger.New(data, meta)
	if err != nil {
		return nil, err
	}
	return grpc.NewWithDB(db, jwtVerifier, maxSizeMsg)
}

// NewWithDB creates a 0-stor server with given DB object
func NewWithDB(db db.DB, jwtVerifier jwt.TokenVerifier, maxSizeMsg int) (api.API, error) {
	return grpc.NewWithDB(db, jwtVerifier, maxSizeMsg)
}
