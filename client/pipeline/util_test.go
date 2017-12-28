package pipeline

import (
	"errors"
	"fmt"
	"net"

	clientGRPC "github.com/zero-os/0-stor/client/datastor/grpc"
	serverGRPC "github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db/memory"
)

func newGRPCServerCluster(count int) (*clientGRPC.Cluster, func(), error) {
	if count < 1 {
		return nil, nil, errors.New("invalid GRPC server-client count")
	}
	var (
		cleanupSlice []func()
		addressSlice []string
	)
	for i := 0; i < count; i++ {
		_, addr, cleanup, err := newGRPCServerClient()
		if err != nil {
			for _, cleanup := range cleanupSlice {
				cleanup()
			}
			return nil, nil, err
		}
		cleanupSlice = append(cleanupSlice, cleanup)
		addressSlice = append(addressSlice, addr)
	}

	cluster, err := clientGRPC.NewCluster(addressSlice, "myLabel", nil)
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

func newGRPCServerClient() (*clientGRPC.Client, string, func(), error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, "", nil, err
	}

	server, err := serverGRPC.New(memory.New(), nil, 0, 0)
	if err != nil {
		return nil, "", nil, err
	}
	go func() {
		err := server.Serve(listener)
		if err != nil {
			panic(err)
		}
	}()

	client, err := clientGRPC.NewClient(listener.Addr().String(), "myLabel", nil)
	if err != nil {
		server.Close()
		return nil, "", nil, err
	}

	clean := func() {
		fmt.Sprintln("clean called")
		err := client.Close()
		if err != nil {
			panic(err)
		}
		err = server.Close()
		if err != nil {
			panic(err)
		}
	}

	return client, listener.Addr().String(), clean, nil
}
