package grpc

import (
	"errors"
	"fmt"

	"github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db/memory"
)

func newServerCluster(count int) (*Cluster, func(), error) {
	if count < 1 {
		return nil, nil, errors.New("invalid GRPC server-client count")
	}
	var (
		cleanupSlice []func()
		addressSlice []string
	)
	for i := 0; i < count; i++ {
		_, addr, cleanup, err := newServerClient()
		if err != nil {
			for _, cleanup := range cleanupSlice {
				cleanup()
			}
			return nil, nil, err
		}
		cleanupSlice = append(cleanupSlice, cleanup)
		addressSlice = append(addressSlice, addr)
	}

	cluster, err := NewCluster(addressSlice, "myLabel", nil)
	if err != nil {
		for _, cleanup := range cleanupSlice {
			cleanup()
		}
		return nil, nil, err
	}

	cleanup := func() {
		cluster.Close()
		for _, cleanup := range cleanupSlice {
			cleanup()
		}
	}
	return cluster, cleanup, nil
}

func newServer() (string, func(), error) {
	server, err := grpc.New(memory.New(), nil, 0, 0)
	if err != nil {
		return "", nil, err
	}
	go func() {
		err := server.Listen("localhost:0")
		if err != nil {
			panic(err)
		}
	}()
	return server.Address(), server.Close, nil
}

func newServerClient() (*Client, string, func(), error) {
	addr, cleanup, err := newServer()
	if err != nil {
		return nil, "", nil, err
	}

	client, err := NewClient(addr, "myLabel", nil)
	if err != nil {
		cleanup()
		return nil, "", nil, err
	}

	clean := func() {
		fmt.Sprintln("clean called")
		client.Close()
		cleanup()
	}

	return client, addr, clean, nil
}
