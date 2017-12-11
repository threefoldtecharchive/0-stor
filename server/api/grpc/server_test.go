package grpc

import (
	"crypto/rand"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db/memory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestServerMsgSize(t *testing.T) {
	require := require.New(t)

	mib := 1024 * 1024

	for i := 2; i <= 64; i *= 4 {
		t.Run(fmt.Sprintf("size %d", i), func(t *testing.T) {
			maxSize := i
			srv, err := New(memory.New(), nil, maxSize, 0)
			require.NoError(err, "server should have been created")
			defer srv.Close()

			go func() {
				err := srv.Listen("localhost:0")
				require.NoError(err, "server should have started listening")
			}()

			cl, err := storgrpc.NewClient(srv.Address(), "testnamespace", "")
			require.NoError(err, "client should have been created")

			key := []byte("foo")

			bigData := make([]byte, (maxSize*mib)+10)
			_, err = rand.Read(bigData)
			require.NoError(err, "should have read random data")

			smallData := make([]byte, (maxSize/2)*mib)
			_, err = rand.Read(smallData)
			require.NoError(err, "should have read random data")

			err = cl.SetObject(datastor.Object{
				Key:  key,
				Data: bigData,
			})
			require.Error(err, "should have exceeded message max size")

			err = cl.SetObject(datastor.Object{
				Key:  key,
				Data: smallData,
			})
			require.NoError(err, "should not have exceeded message max size")

			status, err := cl.GetObjectStatus(key)
			require.NoError(err, "object should exist")
			require.Equal(datastor.ObjectStatusOK, status, "object should exists")

			obj, err := cl.GetObject(key)
			require.NoError(err, "should be able to read message")
			require.Equal(smallData, obj.Data)
		})
	}
}

func TestServerListObjectKeys(t *testing.T) {
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
	jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
		Read: true,
	})
	require.NoError(err, "fail to generate jwt")
	t.Run("valid object", func(t *testing.T) {
		ctx := contextWithToken(nil, jwt)
		stream, err := cl.ListObjectKeys(ctx, &pb.ListObjectKeysRequest{})
		require.NoError(err)
		_, err = stream.Recv()
		requireGRPCError(t, rpctypes.ErrNilLabel, err)
		require.NoError(stream.CloseSend())

		ctx = contextWithLabelAndToken(nil, jwt, label)
		stream, err = cl.ListObjectKeys(ctx, &pb.ListObjectKeysRequest{})
		require.NoError(err, "can't send list request to server")

		objNr := 0
		for {
			obj, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error while reading stream: %v", err)
			}

			objNr++
			key := obj.GetKey()
			_, ok := bufList[string(key)]
			require.True(ok, fmt.Sprintf("received key that was not present in db %s", key))
		}
		assert.Equal(len(bufList), objNr)
	})

	t.Run("wrong permission", func(t *testing.T) {
		jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
			Write: true,
		})
		require.NoError(err, "fail to generate jwt")

		ctx := contextWithLabelAndToken(nil, jwt, label)

		stream, err := cl.ListObjectKeys(ctx, &pb.ListObjectKeysRequest{})
		require.NoError(err, "failed to call List")

		_, err = stream.Recv()
		if err == io.EOF {
		}

		require.Error(err)
		err = rpctypes.Error(err)
		assert.Equal(rpctypes.ErrPermissionDenied, err)
	})

	t.Run("admin right", func(t *testing.T) {
		jwt, err := iyoCl.CreateJWT(namespace, itsyouonline.Permission{
			Admin: true,
		})
		require.NoError(err, "fail to generate jwt")

		ctx := contextWithLabelAndToken(nil, jwt, label)

		stream, err := cl.ListObjectKeys(ctx, &pb.ListObjectKeysRequest{})
		require.NoError(err, "failed to call List")
		_, err = stream.Recv()
		assert.NoError(err)
		stream.CloseSend()
	})
}

func TestServerGetObjectStatus(t *testing.T) {
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
		expectedStatus pb.ObjectStatus
	}{
		{
			name:           "valid",
			keys:           []string{"testkey1", "testkey2", "testkey3"},
			expectedStatus: pb.ObjectStatusOK,
		},
		{
			name:           "missing",
			keys:           []string{"dontexsits"},
			expectedStatus: pb.ObjectStatusMissing,
		},
	}

	for _, tc := range tt {
		ctx := contextWithLabelAndToken(nil, jwt, label)

		for _, key := range tc.keys {
			resp, err := cl.GetObjectStatus(ctx,
				&pb.GetObjectStatusRequest{Key: []byte(key)})
			require.NoError(err, "fail to send request")
			assert.Equal(tc.expectedStatus, resp.GetStatus(), fmt.Sprintf("status should be %v", tc.expectedStatus))
		}
	}
}

func TestServerUpdateReferenceList(t *testing.T) {
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
	ctx := contextWithLabelAndToken(nil, jwt, label)

	_, err = cl.SetReferenceList(ctx, &pb.SetReferenceListRequest{
		Key:           testKey,
		ReferenceList: []string{"ref1"},
	})
	require.NoError(err)

	curReflist := getCurrentReflist(ctx, require, cl, testKey, label)
	require.Equal([]string{"ref1"}, curReflist)

	// append reflist with "ref2"
	_, err = cl.AppendToReferenceList(ctx, &pb.AppendToReferenceListRequest{
		Key:           testKey,
		ReferenceList: []string{"ref2"},
	})
	require.NoError(err)

	curReflist = getCurrentReflist(ctx, require, cl, testKey, label)
	require.Equal([]string{"ref1", "ref2"}, curReflist)

	// remove "ref2"
	resp, err := cl.DeleteFromReferenceList(ctx, &pb.DeleteFromReferenceListRequest{
		Key:           testKey,
		ReferenceList: []string{"ref2"},
	})
	require.NoError(err)
	require.Equal(int64(1), resp.GetCount())

	curReflist = getCurrentReflist(ctx, require, cl, testKey, label)
	require.Equal([]string{"ref1"}, curReflist)

	// remove "ref1"
	resp, err = cl.DeleteFromReferenceList(ctx, &pb.DeleteFromReferenceListRequest{
		Key:           testKey,
		ReferenceList: []string{"ref1"},
	})
	require.NoError(err)
	require.Equal(int64(0), resp.GetCount())

	curReflist = getCurrentReflist(ctx, require, cl, testKey, label)
	require.Empty(curReflist)
}

func getCurrentReflist(ctx context.Context, require *require.Assertions, cl pb.ObjectManagerClient, key []byte, label string) []string {
	resp, err := cl.GetReferenceList(ctx, &pb.GetReferenceListRequest{
		Key: key,
	})
	if err != nil {
		err = rpctypes.Error(err)
		if err == rpctypes.ErrKeyNotFound {
			return nil
		}
		require.Fail(err.Error())
	}
	return resp.GetReferenceList()
}

func contextWithLabelAndToken(ctx context.Context, token, label string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md := metadata.Pairs(rpctypes.MetaAuthKey, token, rpctypes.MetaLabelKey, label)
	return metadata.NewOutgoingContext(ctx, md)
}

func contextWithToken(ctx context.Context, token string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md := metadata.Pairs(rpctypes.MetaAuthKey, token)
	return metadata.NewOutgoingContext(ctx, md)
}
