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

package pipeline

import (
	"github.com/threefoldtech/0-stor/client/datastor"
	"github.com/threefoldtech/0-stor/client/datastor/zerodb"
	zdbtest "github.com/threefoldtech/0-stor/client/datastor/zerodb/test"
)

func newZdbServerCluster(count int) (clu *zerodb.Cluster, cleanup func(), err error) {
	var (
		addresses []string
		cleanups  []func()
		addr      string
	)

	const (
		namespace = "ns"
		passwd    = "passwd"
	)

	for i := 0; i < count; i++ {
		_, addr, cleanup, err = zdbtest.NewInMem0DBServer(namespace)
		if err != nil {
			return
		}
		cleanups = append(cleanups, cleanup)
		addresses = append(addresses, addr)
	}

	clu, err = zerodb.NewCluster(addresses, passwd, namespace, nil, datastor.SpreadingTypeRandom)
	if err != nil {
		return
	}

	cleanup = func() {
		clu.Close()
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
	return

}

func newZdbServerClient(passwd, namespace string) (cli *zerodb.Client, addr string, cleanup func(), err error) {
	var serverCleanup func()

	_, addr, serverCleanup, err = zdbtest.NewInMem0DBServer(namespace)
	if err != nil {
		return
	}
	cli, err = zerodb.NewClient(addr, passwd, namespace)
	if err != nil {
		return
	}

	cleanup = func() {
		serverCleanup()
		cli.Close()
	}
	return
}
