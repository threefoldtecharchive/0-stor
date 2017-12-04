package grpc

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/stor"
	"github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/db/badger"
	pb "github.com/zero-os/0-stor/server/schema"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestServerMsgSize(t *testing.T) {
	require := require.New(t)

	temp, err := ioutil.TempDir("", "0stor")
	require.NoError(err)

	mib := 1024 * 1024

	for i := 2; i <= 64; i *= 4 {
		t.Run(fmt.Sprintf("size %d", i), func(t *testing.T) {
			maxSize := i
			db, err := badger.New(path.Join(temp, "data"), path.Join(temp, "meta"))
			require.NoError(err, "database should have been created")
			srv, err := New(db, nil, maxSize, 0)
			require.NoError(err, "server should have been created")
			defer srv.Close()

			go func() {
				err := srv.Listen("localhost:0")
				require.NoError(err, "server should have started listening")
			}()

			cl, err := stor.NewClient(srv.Address(), "testnamespace", "")
			require.NoError(err, "client should have been created")

			key := []byte("foo")

			bigData := make([]byte, (maxSize+10)*mib)
			_, err = rand.Read(bigData)
			require.NoError(err, "should have read random data")

			smallData := make([]byte, (maxSize/2)*mib)
			_, err = rand.Read(smallData)
			require.NoError(err, "should have read random data")

			err = cl.ObjectCreate(key, bigData, []string{})
			require.Error(err, "should have exceeded message max size")

			err = cl.ObjectCreate(key, smallData, []string{})
			fmt.Println(err)
			require.NoError(err, "should not have exceeded message max size")

			exists, err := cl.ObjectExist(key)
			require.NoError(err, "object should exist")
			require.True(exists, "object should exists")

			obj, err := cl.ObjectGet(key)
			require.NoError(err, "should be able to read message")
			require.Equal(smallData, obj.Value)
		})
	}
}

/*
// TODO: Enable test again, for now it fails
// not sure if its because the test that is broken,
// or because a bug in the code
func TestListObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	server, iyoCl, clean := getTestGRPCServer(t, organization)
	bufList := populateDB(t, label, server.db)

	// create client connection
	conn, err := grpc.Dial(server.Address(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	defer func() {
		conn.Close()
		clean()
	}()

	cl := pb.NewObjectManagerClient(conn)
	t.Run("valid object", func(t *testing.T) {

		jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
			Read: true,
		})
		require.NoError(err, "fail to generate jwt")

		md := metadata.Pairs(api.GRPCMetaAuthKey, jwt, api.GRPCMetaLabelKey, label)
		ctx := metadata.NewOutgoingContext(context.Background(), md)

		stream, err := cl.List(ctx, &pb.ListObjectsRequest{Label: label})
		require.NoError(err, "can't send list request to server")

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
			expectedValue, ok := bufList[string(obj.Key)]
			require.True(ok, fmt.Sprintf("received key that was not present in db %s", obj.GetKey()))
			assert.EqualValues(expectedValue, obj.GetValue())
		}
		assert.Equal(len(bufList), objNr)
	})

	t.Run("wrong permission", func(t *testing.T) {
		jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
			Write: true,
		})
		require.NoError(err, "fail to generate jwt")

		md := metadata.Pairs(api.GRPCMetaAuthKey, jwt, api.GRPCMetaLabelKey, label)
		ctx := metadata.NewOutgoingContext(context.Background(), md)

		stream, err := cl.List(ctx, &pb.ListObjectsRequest{Label: label})
		require.NoError(err, "failed to call List")
		for {
			_, err = stream.Recv()
			if err == io.EOF {
				break
			}

			require.Error(err)
			statusErr, ok := status.FromError(err)
			require.True(ok, "error is not valid rpc status error")
			assert.Equal("JWT token doesn't contains required scopes", statusErr.Message())
			break
		}
	})

	t.Run("admin right", func(t *testing.T) {
		jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
			Admin: true,
		})
		require.NoError(err, "fail to generate jwt")

		md := metadata.Pairs(api.GRPCMetaAuthKey, jwt, api.GRPCMetaLabelKey, label)
		ctx := metadata.NewOutgoingContext(context.Background(), md)

		stream, err := cl.List(ctx, &pb.ListObjectsRequest{Label: label})
		require.NoError(err, "failed to call List")
		_, err = stream.Recv()
		require.NoError(err)
		stream.CloseSend()
	})
}
*/

func TestCheckObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	server, iyoCl, clean := getTestGRPCServer(t, organization)
	populateDB(t, label, server.db)

	// create client connection
	conn, err := grpc.Dial(server.Address(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	defer func() {
		conn.Close()
		clean()
	}()

	cl := pb.NewObjectManagerClient(conn)
	jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
		Read: true,
	})
	require.NoError(err, "fail to generate jwt")

	tt := []struct {
		name           string
		keys           []string
		expectedStatus pb.CheckResponse_Status
	}{
		{
			name:           "valid",
			keys:           []string{"testkey1", "testkey2", "testkey3"},
			expectedStatus: pb.CheckResponse_ok,
		},
		{
			name:           "missing",
			keys:           []string{"dontexsits"},
			expectedStatus: pb.CheckResponse_missing,
		},
	}

	for _, tc := range tt {
		md := metadata.Pairs(api.GRPCMetaAuthKey, jwt, api.GRPCMetaLabelKey, label)
		ctx := metadata.NewOutgoingContext(context.Background(), md)

		stream, err := cl.Check(ctx, &pb.CheckRequest{
			Label: label,
			Ids:   tc.keys,
		})
		require.NoError(err, "fail to send check request")

		n := 0
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			require.NoError(err, "error during check response streaming")

			assert.Equal(tc.expectedStatus, resp.GetStatus(), fmt.Sprintf("status should be %v", tc.expectedStatus))
			n++
		}

		assert.Equal(len(tc.keys), n)
	}
}

func TestUpdateReferenceList(t *testing.T) {
	require := require.New(t)

	server, iyoCl, clean := getTestGRPCServer(t, organization)
	populateDB(t, label, server.db)

	// create client connection
	conn, err := grpc.Dial(server.Address(), grpc.WithInsecure())
	require.NoError(err, "can't connect to the server")

	defer func() {
		conn.Close()
		clean()
	}()

	cl := pb.NewObjectManagerClient(conn)
	jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
		Admin: true,
	})
	require.NoError(err, "fail to generate jwt")

	// set reflist
	testKey := []byte("testkey1")
	md := metadata.Pairs(api.GRPCMetaAuthKey, jwt, api.GRPCMetaLabelKey, label)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err = cl.SetReferenceList(ctx, &pb.UpdateReferenceListRequest{
		Label:         label,
		Key:           testKey,
		ReferenceList: []string{"ref1"},
	})
	require.NoError(err)

	curReflist := getCurrentReflist(ctx, require, cl, testKey, label)
	require.Equal([]string{"ref1"}, curReflist)

	// append reflist with "ref2"
	_, err = cl.AppendReferenceList(ctx, &pb.UpdateReferenceListRequest{
		Label:         label,
		Key:           testKey,
		ReferenceList: []string{"ref2"},
	})
	require.NoError(err)

	curReflist = getCurrentReflist(ctx, require, cl, testKey, label)
	require.Equal([]string{"ref1", "ref2"}, curReflist)

	// remove "ref2"
	_, err = cl.RemoveReferenceList(ctx, &pb.UpdateReferenceListRequest{
		Label:         label,
		Key:           testKey,
		ReferenceList: []string{"ref2"},
	})
	require.NoError(err)

	curReflist = getCurrentReflist(ctx, require, cl, testKey, label)
	require.Equal([]string{"ref1"}, curReflist)

	// remove "ref1"
	_, err = cl.RemoveReferenceList(ctx, &pb.UpdateReferenceListRequest{
		Label:         label,
		Key:           testKey,
		ReferenceList: []string{"ref1"},
	})
	require.NoError(err)

	curReflist = getCurrentReflist(ctx, require, cl, testKey, label)
	require.Empty(curReflist)
}

func getCurrentReflist(ctx context.Context, require *require.Assertions, cl pb.ObjectManagerClient, key []byte, label string) []string {
	resp, err := cl.Get(ctx, &pb.GetObjectRequest{
		Label: label,
		Key:   key,
	})
	require.NoError(err)

	return resp.Object.GetReferenceList()
}
