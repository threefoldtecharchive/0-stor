package grpc

import (
	"errors"
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/client"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
	"google.golang.org/grpc"
)

var (
	// ErrNilClient returned when user try to create daemon with nil client
	ErrNilClient = errors.New("nil client")
)

type storClient interface {
	objectClient
	namespaceClient
	Close() error
}

// Daemon represents a client daemon
type Daemon struct {
	grpcServer  *grpc.Server
	client      storClient
	listener    net.Listener
	listeningCh chan struct{}
}

// New creates new daemon with given client
// and maximum message size = maxMsgSize MiB
func New(client *client.Client, maxMsgSize int) (*Daemon, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return newDaemon(client, maxMsgSize), nil
}

func newDaemon(client storClient, maxMsgSize int) *Daemon {
	max := mibToBytes(maxMsgSize)

	daemon := &Daemon{
		grpcServer: grpc.NewServer(
			grpc.MaxRecvMsgSize(max),
			grpc.MaxSendMsgSize(max),
		),
		client:      client,
		listeningCh: make(chan struct{}, 1),
	}
	pb.RegisterObjectServiceServer(daemon.grpcServer, newObjectSrv(client))
	pb.RegisterNamespaceServiceServer(daemon.grpcServer, newNamespaceSrv(client))

	return daemon
}

// Listen listens to specified address
func (d *Daemon) Listen(addr string) error {
	list, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	d.listener = list

	log.Infof("daemon listen at : %v", d.listener.Addr().String())

	d.listeningCh <- struct{}{}
	return d.grpcServer.Serve(list)
}

// Close closes the daemon
func (d *Daemon) Close() {
	if d.listener != nil {
		d.listener.Close()
	}
	d.grpcServer.GracefulStop()
	d.client.Close()
}

func mibToBytes(n int) int {
	return n * 1024 * 1024
}
