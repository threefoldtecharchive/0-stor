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

package zerodb

import (
	"github.com/threefoldtech/0-stor/client/datastor"
)

// Shard implements datastor.Shard for
// 0-db clients, to make those clients work within a cluster of other 0-db clients.
type Shard struct {
	*Client
	address string
}

// Identifier implements datastor.Shard.Identifier
func (shard *Shard) Identifier() string {
	return shard.address
}

var (
	_ datastor.Shard = (*Shard)(nil)
)
