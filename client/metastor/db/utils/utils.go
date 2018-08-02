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

package utils

import (
	"fmt"

	"github.com/threefoldtech/0-stor/client/metastor/db"
	"github.com/threefoldtech/0-stor/client/metastor/db/badger"
	"github.com/threefoldtech/0-stor/client/metastor/db/etcd"

	"github.com/mitchellh/mapstructure"
)

// NewMetaStorDB creates new metastor DB implementation.
// `dbType` defines the type of the implementations we want to create.
// `config` is the db configuration.
func NewMetaStorDB(dbType string, config map[string]interface{}) (db.DB, error) {
	switch dbType {
	case db.TypeBadger:
		var badgerConf badger.Config

		err := mapstructure.Decode(config, &badgerConf)
		if err != nil {
			return nil, err
		}
		return badger.New(badgerConf.DataDir, badgerConf.MetaDir)
	case db.TypeETCD:
		var etcdConf etcd.Config

		err := mapstructure.Decode(config, &etcdConf)
		if err != nil {
			return nil, err
		}
		return etcd.New(etcdConf.Endpoints)
	default:
		return nil, fmt.Errorf("invalid db type:%v", dbType)
	}

}
