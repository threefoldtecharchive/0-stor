package grpc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/zero-os/0-stor/server/storserver"

	"github.com/stretchr/testify/require"
)

func getTestGRPCServer(t *testing.T) (storserver.StoreServer, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	server, err := storserver.NewGRPC(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	require.NoError(t, err)

	_, err = server.Listen("localhost:0")
	require.NoError(t, err, "server failed to start listening")

	clean := func() {
		fmt.Sprintln("clean called")
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return server, clean
}
