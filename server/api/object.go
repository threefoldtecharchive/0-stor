package api

import (
	"github.com/zero-os/0-stor/server"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/encoding"
)

// CheckStatusForObject allows you to check the status for a given object in a given database.
// This function ensures that the object's data exist and is not corrupt.
// If the object has a reference list stored, it is also checked if that one is not corrupt.
func CheckStatusForObject(namespace, key []byte, db dbp.DB) (server.CheckStatus, error) {
	// validate data of object
	dataKey := dbp.DataKey(namespace, key)
	status, err := checkStatusForBlob(dataKey, db, false)
	if err != nil || status != server.CheckStatusOK {
		return status, err
	}

	// data is valid, so let's check/validate the reference list
	// (if it exist for this object)
	refListKey := dbp.ReferenceListKey(namespace, key)
	return checkStatusForBlob(refListKey, db, true)
}

func checkStatusForBlob(key []byte, db dbp.DB, optional bool) (server.CheckStatus, error) {
	// see if we can fetch the blob's package
	data, err := db.Get(key)
	if err != nil {
		if err == dbp.ErrNotFound {
			if optional {
				return server.CheckStatusOK, nil
			}
			return server.CheckStatusMissing, nil
		}
		return server.CheckStatus(0), err
	}

	// validate the blob
	err = encoding.ValidateData(data)
	if err != nil {
		return server.CheckStatusCorrupted, nil
	}

	// blob exists and is valid
	return server.CheckStatusOK, nil
}
