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

package test

import (
	"fmt"
	"strings"
	"sync"

	"github.com/iwanbk/redcon" // using this because we have race condition issue with upstream
)

type inMem0DBServer struct {
	mu        sync.RWMutex
	items     map[string][]byte
	server    *redcon.Server
	namespace string
	counter   int
}

func NewInMem0DBServer(namespace string) (string, func(), error) {
	s := &inMem0DBServer{
		items:     make(map[string][]byte),
		namespace: namespace,
	}
	s.server = redcon.NewServer("localhost:0", s.handler, s.accept, s.closeHandler)

	if err := s.start(); err != nil {
		return "", nil, err
	}
	cleanup := func() {
		s.Close()
	}

	return s.server.ListenAddress(), cleanup, nil
}

func (s *inMem0DBServer) start() error {
	errCh := make(chan error)
	go s.server.ListenServeAndSignal(errCh)
	err := <-errCh
	return err
}

func (s *inMem0DBServer) Close() error {
	return s.server.Close()
}

func (s *inMem0DBServer) handler(conn redcon.Conn, cmd redcon.Command) {
	switch strings.ToLower(string(cmd.Args[0])) {
	case "select":
		conn.WriteString("OK")
	case "set":
		s.set(conn, cmd)
	case "get":
		s.get(conn, cmd)
	case "exist":
		s.exist(conn, cmd)
	case "check":
		s.check(conn, cmd)
	case "del":
		s.del(conn, cmd)
	case "nsinfo":
		s.nsinfo(conn, cmd)
	case "quit":
		conn.WriteString("OK")
		conn.Close()
	default:
		conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
	}
}

// TODO : also support `corrupted`
func (s *inMem0DBServer) check(conn redcon.Conn, cmd redcon.Command) {
	s.mu.Lock()

	_, ok := s.items[string(cmd.Args[1])]
	s.mu.Unlock()

	if ok {
		conn.WriteInt(1)
	} else {
		conn.WriteInt(-1)
	}
}
func (s *inMem0DBServer) set(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter += 1
	key := fmt.Sprintf("key-%d", s.counter)
	s.items[key] = cmd.Args[2]

	conn.WriteBulk([]byte(key))
}
func (s *inMem0DBServer) exist(conn redcon.Conn, cmd redcon.Command) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.items[string(cmd.Args[1])]
	if ok {
		conn.WriteInt(1)
	} else {
		conn.WriteInt(0)
	}
}
func (s *inMem0DBServer) del(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	key := string(cmd.Args[1])

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.items[key]
	delete(s.items, key)

	if !ok {
		conn.WriteInt(0)
	} else {
		conn.WriteInt(1)
	}

}

func (s *inMem0DBServer) get(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.items[string(cmd.Args[1])]

	if !ok {
		conn.WriteNull()
	} else {
		conn.WriteBulk(val)
	}
}

func (s *inMem0DBServer) nsinfo(conn redcon.Conn, cmd redcon.Command) {
	format := "# namespace\nname: %v\nentries: %v\npublic: yes\npassword: no\ndata_size_bytes: 24\ndata_size_mb: 0.00\ndata_limits_bytes: 0\nindex_size_bytes: 324\nindex_size_kb: 0.32\n"
	s.mu.RLock()
	defer s.mu.RUnlock()

	str := fmt.Sprintf(format, s.namespace, len(s.items))
	conn.WriteBulk([]byte(str))
}
func (s *inMem0DBServer) accept(conn redcon.Conn) bool {
	return true
}
func (s *inMem0DBServer) closeHandler(conn redcon.Conn, err error) {
}
