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

package grpc

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/encoding"

	"github.com/stretchr/testify/require"
)

func TestNewNamespaceAPIPanics(t *testing.T) {
	require.Panics(t, func() {
		NewNamespaceAPI(nil)
	}, "no db given")
}

func TestGetNamespace(t *testing.T) {
	require := require.New(t)

	api, clean := getTestNamespaceAPI(t)
	defer clean()

	data, err := encoding.EncodeNamespace(server.Namespace{Label: []byte(label)})
	require.NoError(err)
	err = api.db.Set(db.NamespaceKey([]byte(label)), data)
	require.NoError(err)

	req := &pb.GetNamespaceRequest{}

	resp, err := api.GetNamespace(context.Background(), req)
	require.Error(err)
	err = rpctypes.Error(err)
	require.Equal(rpctypes.ErrNilLabel, err)

	resp, err = api.GetNamespace(contextWithLabel(nil, label), req)
	require.NoError(err)

	require.Equal(label, resp.GetLabel())
	require.EqualValues(0, resp.GetNrObjects())
}

func getTestNamespaceAPI(t *testing.T) (*NamespaceAPI, func()) {
	require := require.New(t)

	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(err)
	}

	clean := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	require.NoError(err)
	api := NewNamespaceAPI(db)

	return api, clean
}
