/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"github.com/zero-os/0-stor/server"
	dbp "github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/encoding"
)

// ObjectStatusForObject allows you to check the status
// for a given object in a given database.
// This function ensures that the object's data exist and is not corrupt.
func ObjectStatusForObject(key []byte, db dbp.DB) (server.ObjectStatus, error) {
	// see if we can fetch the object's data package
	data, err := db.Get(key)
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
