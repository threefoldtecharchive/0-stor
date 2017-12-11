package api

import (
	"github.com/zero-os/0-stor/server"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/encoding"
)

// ObjectStatusForObject allows you to check the status
// for a given object in a given database.
// This function ensures that the object's data exist and is not corrupt.
// If the object has a reference list stored, it is also checked
// if that one is not corrupt.
func ObjectStatusForObject(namespace, key []byte, db dbp.DB) (server.ObjectStatus, error) {
	// validate data of object
	dataKey := dbp.DataKey(namespace, key)
	status, err := objectStatusForBlob(dataKey, db, false)
	if err != nil || status != server.ObjectStatusOK {
		return status, err
	}

	// data is valid, so let's check/validate the reference list
	// (if it exist for this object)
	refListKey := dbp.ReferenceListKey(namespace, key)
	return objectStatusForBlob(refListKey, db, true)
}

func objectStatusForBlob(key []byte, db dbp.DB, optional bool) (server.ObjectStatus, error) {
	// see if we can fetch the blob's package
	data, err := db.Get(key)
	if err != nil {
		if err == dbp.ErrNotFound {
			if optional {
				return server.ObjectStatusOK, nil
			}
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
