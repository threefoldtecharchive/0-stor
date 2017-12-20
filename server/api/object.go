package api

import (
	"github.com/zero-os/0-stor/server"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/encoding"
)

// ObjectStatusForObject allows you to check the status
// for a given object in a given database.
// This function ensures that the object's data exist and is not corrupt.
func ObjectStatusForObject(namespace, key []byte, db dbp.DB) (server.ObjectStatus, error) {
	dataKey := dbp.DataKey(namespace, key)
	// see if we can fetch the object's data package
	data, err := db.Get(dataKey)
	if err != nil {
		if err == dbp.ErrNotFound {
			return server.ObjectStatusMissing, nil
		}
		return server.ObjectStatus(0), err
	}

	// validate the blob
	err = encoding.ValidateData(data)
	if err != nil {
		return server.ObjectStatusCorrupted, nil
	}

	// blob exists and is valid
	return server.ObjectStatusOK, nil
}
