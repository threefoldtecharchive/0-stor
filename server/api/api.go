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
	"net"
)

// Server defines the 0-stor Server API interface.
type Server interface {
	// Serve accepts incoming connections on the listener, lis.
	// This function blocks until the given listener, list, is closed.
	// The given listener, lis, is owned by the Server as soon as this function is called,
	// and the server will close any active listeners as part of its GracefulStop method.
	Serve(lis net.Listener) error

	// Close closes the 0-stor server its resources and stops all it open connections gracefully.
	// It stops the server from accepting new connections and blocks until
	// all established connections and other resources have been closed.
	Close() error
}
