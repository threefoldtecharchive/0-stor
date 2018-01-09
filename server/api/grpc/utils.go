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
	log "github.com/Sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

// CodeToLogrusLevel is the default 0-stor implementation
// of gRPC return codes to log(rus) levels for server side.
func CodeToLogrusLevel(code codes.Code) log.Level {
	if level, ok := _GRPCCodeToLogrusLevelMapping[code]; ok {
		return level
	}
	return log.ErrorLevel
}

var _GRPCCodeToLogrusLevelMapping = map[codes.Code]log.Level{
	codes.OK:                 log.DebugLevel,
	codes.Canceled:           log.DebugLevel,
	codes.InvalidArgument:    log.DebugLevel,
	codes.NotFound:           log.DebugLevel,
	codes.AlreadyExists:      log.DebugLevel,
	codes.Unauthenticated:    log.InfoLevel,
	codes.PermissionDenied:   log.InfoLevel,
	codes.DeadlineExceeded:   log.WarnLevel,
	codes.ResourceExhausted:  log.WarnLevel,
	codes.FailedPrecondition: log.WarnLevel,
	codes.Aborted:            log.WarnLevel,
}
