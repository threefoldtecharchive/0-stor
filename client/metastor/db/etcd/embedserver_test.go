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

package etcd

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/coreos/etcd/embed"
	"github.com/coreos/pkg/capnslog"
)

// Server is embedded metadata server
// which listen on unix socket
type Server struct {
	lcDir      string
	lpDir      string
	etcd       *embed.Etcd
	listenAddr string
}

// Stop stops the server and release it's resources
func (s *Server) Stop() {
	s.etcd.Server.Stop()
	<-s.etcd.Server.StopNotify()
	s.etcd.Close()
	os.RemoveAll(s.lpDir)
	os.RemoveAll(s.lcDir)
}

// ListenAddrs returns listen address of this server
func (s *Server) ListenAddr() string {
	return s.listenAddr
}

// New creates new embedded metadata server
func NewEmbeddedServer() (*Server, error) {
	tmpDir, err := ioutil.TempDir("", "etcd")
	if err != nil {
		return nil, err
	}

	cfg := embed.NewConfig()
	cfg.Dir = tmpDir

	capnslog.SetGlobalLogLevel(capnslog.CRITICAL)
	// listen client URL
	// we use tmpDir as unix address because it is a simple
	// yet valid way to generate random string
	lcurl, err := url.Parse("unix://" + filepath.Base(tmpDir))
	if err != nil {
		return nil, err
	}
	cfg.LCUrls = []url.URL{*lcurl}

	// listen peer url
	// same strategy with listen client URL
	lpDir, err := ioutil.TempDir("", "etcd")
	if err != nil {
		return nil, err
	}
	lpurl, err := url.Parse("unix://" + filepath.Base(lpDir))
	if err != nil {
		return nil, err
	}
	cfg.LPUrls = []url.URL{*lpurl}

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		return nil, err
	}

	<-e.Server.ReadyNotify()

	conf := e.Config()

	return &Server{
		lpDir:      lpDir,
		lcDir:      tmpDir,
		etcd:       e,
		listenAddr: conf.LCUrls[0].String(),
	}, nil
}
