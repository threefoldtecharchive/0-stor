package components

import (
	"fmt"
	"strings"
	"sync"
)

const (
	// ShardType0Stor means that it is 0-stor shard
	ShardType0Stor = "0-stor"
	// ShardTypeEtcd means that it is etcd shard
	ShardTypeEtcd = "etcd"
)

const (
	// StatusUnknownError means that we don't know exactly what the error is
	StatusUnknownError = 400

	// StatusTimeoutError means that the operation to that shard got timed out
	StatusTimeoutError = 401

	// StatusInvalidShardAddress means that shard address is invalid
	// which make us unable to creates the client for it.
	// It could only happen if the metadata is invalid
	StatusInvalidShardAddress = 402
)

// ServerError represents errors at the server
type ServerError struct {
	Addrs []string `json:"addrs"`
	Kind  string   `json:"kind"`
	Err   error    `json:"error"`
	Code  int      `json:"code"`
}

// Error implements error interface
func (se *ServerError) Error() string {
	if len(se.Addrs) == 0 {
		return fmt.Sprintf("server(s) %s errored with: %v", se.Kind, se.Err)
	}
	return fmt.Sprintf("server(s) %s %s: errored with: %v",
		se.Kind, strings.Join(se.Addrs, ","), se.Err)
}

// ShardError represents errors in a shard
// It implements error interface on which the `Error` func
// returns JSON string which make it easy to be parsed by other module
// or language
type ShardError struct {
	errors []ServerError
	mux    sync.Mutex
}

// Add adds an error, it is goroutine safe
func (se *ShardError) Add(addrs []string, kind string, err error, code int) {
	se.mux.Lock()
	defer se.mux.Unlock()

	if code == 0 {
		code = StatusUnknownError
	}
	se.errors = append(se.errors, ServerError{
		Addrs: addrs,
		Kind:  kind,
		Err:   err,
		Code:  code,
	})
}

// Nil returns true if it is a nil error
func (se *ShardError) Nil() bool {
	se.mux.Lock()
	defer se.mux.Unlock()

	return len(se.errors) == 0
}

// Num returns the number of underlying errors
func (se *ShardError) Num() int {
	se.mux.Lock()
	defer se.mux.Unlock()

	return len(se.errors)
}

// Error implements error interface
func (se *ShardError) Error() string {
	se.mux.Lock()
	defer se.mux.Unlock()

	if len(se.errors) == 0 {
		return ""
	}

	errStr := "following shard errors occurred:\n"
	for _, err := range se.errors {
		errStr += "\t" + err.Error() + "\n"
	}
	return errStr[:len(errStr)-1]
}

// Errors returns the underlying errors
func (se *ShardError) Errors() []ServerError {
	se.mux.Lock()
	defer se.mux.Unlock()

	return se.errors
}
