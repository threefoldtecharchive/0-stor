package grpc

import (
	"context"
	"fmt"
	"io"
	"testing"

	pb "github.com/zero-os/0-stor/grpc_store"
	"github.com/zero-os/0-stor/server/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestListObject(t *testing.T) {
	server, clean := getTestGRPCServer(t)

	label, bufList := test.PopulateDB(t, server.DB())

	// create client connection
	conn, err := grpc.Dial(server.Addr(), grpc.WithInsecure())
	require.NoError(t, err, "can't connect to the server")

	defer func() {
		conn.Close()
		fmt.Println("defer clean")
		clean()
	}()

	cl := pb.NewObjectManagerClient(conn)
	t.Run("valid object", func(t *testing.T) {

		stream, err := cl.List(context.Background(), &pb.ListObjectsRequest{Label: label})
		require.NoError(t, err, "can't send list request to server")

		objNr := 0
		for i := 0; ; i++ {
			obj, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error while reading stream: %v", err)
			}

			objNr++
			key := fmt.Sprintf("testkey%d", i)
			assert.Equal(t, key, obj.GetKey())
			assert.Equal(t, bufList[i], obj.GetValue())
		}
		assert.Equal(t, objNr, len(bufList))
	})
}
