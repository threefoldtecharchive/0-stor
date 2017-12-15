package grpc

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
)

func TestDaemonNilClient(t *testing.T) {
	_, err := New(nil, 1)
	require.Equal(t, ErrNilClient, err)
}
func TestDaemonMsgSize(t *testing.T) {
	const (
		maxMsgSize = 2
	)
	client := &clientStub{
		objClientStub:       newObjClientStub(),
		namespaceClientStub: &namespaceClientStub{},
	}

	d := newDaemon(client, maxMsgSize)
	go d.Listen("127.0.0.1:0")
	<-d.listeningCh
	defer d.Close()

	// create conn
	conn, err := grpc.Dial(d.listener.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err, "can't connect to daemon")

	// create obj client
	cl := pb.NewObjectServiceClient(conn)

	// test write with data  less than maxMsgSize
	less := make([]byte, mibToBytes(maxMsgSize)-100)
	rand.Read(less)
	_, err = cl.Write(context.Background(), &pb.WriteRequest{
		Key:   []byte("less"),
		Value: less,
	})
	require.NoError(t, err)

	// test write with data  > maxMsgSize
	more := make([]byte, mibToBytes(maxMsgSize)+1)
	rand.Read(more)
	_, err = cl.Write(context.Background(), &pb.WriteRequest{
		Key:   []byte("less"),
		Value: more,
	})
	require.Error(t, err)

}

// test that the daemon has properly set the object service
func TestDaemonObject(t *testing.T) {
	const (
		maxMsgSize = 1
	)
	client := &clientStub{
		objClientStub:       newObjClientStub(),
		namespaceClientStub: &namespaceClientStub{},
	}

	d := newDaemon(client, maxMsgSize)
	go d.Listen("127.0.0.1:0")
	<-d.listeningCh
	defer d.Close()

	// create conn
	conn, err := grpc.Dial(d.listener.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err, "can't connect to daemon")

	// create obj client
	cl := pb.NewObjectServiceClient(conn)

	// test read
	_, err = cl.Write(context.Background(), &pb.WriteRequest{
		Key:   []byte("myKey"),
		Value: []byte("myValue"),
	})
	require.NoError(t, err)
}

// test that the daemon has properly set the namespace service
func TestDaemonNamespace(t *testing.T) {
	const (
		maxMsgSize = 1
	)
	client := &clientStub{
		objClientStub:       newObjClientStub(),
		namespaceClientStub: &namespaceClientStub{},
	}

	d := newDaemon(client, maxMsgSize)
	go d.Listen("127.0.0.1:0")
	<-d.listeningCh
	defer d.Close()

	// create conn
	conn, err := grpc.Dial(d.listener.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err, "can't connect to daemon")

	// create obj client
	cl := pb.NewNamespaceServiceClient(conn)

	// test read
	_, err = cl.CreateNamespace(context.Background(), &pb.NamespaceRequest{
		Namespace: "ns",
	})
	require.NoError(t, err)
}

type clientStub struct {
	*objClientStub
	*namespaceClientStub
}

func (cs *clientStub) Close() error {
	return nil
}
