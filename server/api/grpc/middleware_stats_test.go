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
	"testing"

	"github.com/zero-os/0-stor/server/stats"

	"github.com/stretchr/testify/assert"
)

func TestGetStatsfunc(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		grpcMethod string
		statsFunc  labelStatsFunc
		err        bool
	}{
		{"/ObjectManager/GetObject", stats.IncrRead, false},
		{"/ObjectManager/GetObjectStatus", stats.IncrRead, false},
		{"/ObjectManager/ListObjectKeys", stats.IncrRead, false},
		{"/ObjectManager/CreateObject", stats.IncrWrite, false},
		{"/ObjectManager/DeleteObject", stats.IncrWrite, false},
		{"/NamespaceManager/GetNamespace", stats.IncrRead, false},
		{"", nil, true},
		{"/ObjectManager/", nil, true},
		{"/NamespaceManager/", nil, true},
		{"/ObjectManager/Foo", nil, true},
		{"/NamespaceManager/Bar", nil, true},
	}

	for _, c := range cases {
		statsFunc, err := getStatsFunc(c.grpcMethod)
		if c.err {
			assert.Error(err)
		} else {
			assert.NoError(err)
			r1, w1 := stats.Rate(label)
			statsFunc(label)
			r2, w2 := stats.Rate(label)

			if r1 == r2 && w1 == w2 {
				assert.FailNow("Stats have not changed after statsFunc call")
			}
		}
	}
}
