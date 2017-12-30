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
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/stats"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

var _ (pb.NamespaceManagerServer) = (*NamespaceAPI)(nil)

// NamespaceAPI implements pb.NamespaceManagerServer
type NamespaceAPI struct {
	db db.DB
}

// NewNamespaceAPI returns a NamespaceAPI
func NewNamespaceAPI(db db.DB) *NamespaceAPI {
	if db == nil {
		panic("no database given to NamespaceAPI")
	}

	return &NamespaceAPI{
		db: db,
	}
}

// GetNamespace implements NamespaceManagerServer.GetNamespace
func (api *NamespaceAPI) GetNamespace(ctx context.Context, req *pb.GetNamespaceRequest) (*pb.GetNamespaceResponse, error) {
	label, err := extractStringFromContext(ctx, rpctypes.MetaLabelKey)
	if err != nil {
		return nil, rpctypes.ErrGRPCNilLabel
	}

	count, err := db.CountKeys(api.db, []byte(label))
	if err != nil {
		log.Errorf("Database error for key %v: %v", label, err)
		return nil, rpctypes.ErrGRPCDatabase
	}
	read, write := stats.Rate(label)

	resp := &pb.GetNamespaceResponse{
		Label:               label,
		ReadRequestPerHour:  read,
		WriteRequestPerHour: write,
		NrObjects:           int64(count),
	}

	return resp, nil
}
