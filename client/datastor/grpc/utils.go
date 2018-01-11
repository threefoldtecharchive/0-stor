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
	"net"
	"strings"
	"time"
)

// Dial the underlying connection of a GRPC connection.
// If the given addr contains '/' and/or no colon, a UNIX network is assumed and used,
// otherwise a TCP network is assumed and used.
func Dial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(parseNetworkProtocol(addr), addr, timeout)
}

func parseNetworkProtocol(addr string) string {
	if i := strings.IndexAny(addr, "/:"); i == -1 || addr[i] == '/' {
		return "unix"
	}
	return "tcp"
}
