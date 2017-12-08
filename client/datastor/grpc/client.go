package grpc

import (
	"context"
	"io"
	"math"

	"golang.org/x/sync/errgroup"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

var _ (datastor.Client) = (*Client)(nil)

// Client defines a data client,
// to connect to a zstordb using the GRPC interface.
type Client struct {
	conn             *grpc.ClientConn
	objService       pb.ObjectManagerClient
	namespaceService pb.NamespaceManagerClient

	jwtToken        string
	jwtTokenDefined bool
	label           string
}

// NewClient create a new data client,
// which allows you to connect to a zstordb using a GRPC interface.
// The addres to the zstordb server is required,
// and so is the label, as the latter serves as the identifier of the to be used namespace.
// The jwtToken is required, only if the connected zstordb server requires this.
func NewClient(addr, label, jwtToken string) (*Client, error) {
	if len(addr) == 0 {
		panic("no zstordb address give")
	}
	if len(label) == 0 {
		panic("no label given")
	}

	conn, err := grpc.Dial(addr,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt32),
			grpc.MaxCallSendMsgSize(math.MaxInt32),
		))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:             conn,
		objService:       pb.NewObjectManagerClient(conn),
		namespaceService: pb.NewNamespaceManagerClient(conn),
		jwtToken:         jwtToken,
		jwtTokenDefined:  len(jwtToken) != 0,
		label:            label,
	}, nil
}

// SetObject implements datastor.Client.SetObject
func (c *Client) SetObject(object datastor.Object) error {
	_, err := c.objService.SetObject(c.contextWithMetadata(nil), &pb.SetObjectRequest{
		Key:           object.Key,
		Data:          object.Data,
		ReferenceList: object.ReferenceList,
	})
	if err != nil {
		return toErr(err)
	}
	return nil
}

// GetObject implements datastor.Client.GetObject
func (c *Client) GetObject(key []byte) (*datastor.Object, error) {
	resp, err := c.objService.GetObject(c.contextWithMetadata(nil),
		&pb.GetObjectRequest{Key: key})
	if err != nil {
		return nil, toErr(err)
	}

	dataObject := &datastor.Object{
		Key:           key,
		Data:          resp.GetData(),
		ReferenceList: resp.GetReferenceList(),
	}
	if len(dataObject.Data) == 0 {
		return nil, datastor.ErrMissingData
	}
	return dataObject, nil
}

// DeleteObject implements datastor.Client.DeleteObject
func (c *Client) DeleteObject(key []byte) error {
	// delete the objects from the server
	_, err := c.objService.DeleteObject(
		c.contextWithMetadata(nil), &pb.DeleteObjectRequest{Key: key})
	if err != nil {
		return toErr(err)
	}
	return err
}

// GetObjectStatus implements datastor.Client.GetObjectStatus
func (c *Client) GetObjectStatus(key []byte) (datastor.ObjectStatus, error) {
	resp, err := c.objService.GetObjectStatus(
		c.contextWithMetadata(nil), &pb.GetObjectStatusRequest{Key: key})
	if err != nil {
		return datastor.ObjectStatus(0), toErr(err)
	}
	return convertStatus(resp.GetStatus())
}

// ExistObject implements datastor.Client.ExistObject
func (c *Client) ExistObject(key []byte) (bool, error) {
	status, err := c.GetObjectStatus(key)
	if err != nil {
		return false, err
	}
	switch status {
	case datastor.ObjectStatusOK:
		return true, nil
	case datastor.ObjectStatusCorrupted:
		return false, datastor.ErrObjectCorrupted
	default:
		return false, nil
	}
}

// GetNamespace implements datastor.Client.GetNamespace
func (c *Client) GetNamespace() (*datastor.Namespace, error) {
	resp, err := c.namespaceService.GetNamespace(
		c.contextWithMetadata(nil), &pb.GetNamespaceRequest{})
	if err != nil {
		return nil, toErr(err)
	}

	ns := &datastor.Namespace{Label: resp.GetLabel()}
	if ns.Label != c.label {
		return nil, datastor.ErrInvalidLabel
	}

	ns.ReadRequestPerHour = resp.GetReadRequestPerHour()
	ns.WriteRequestPerHour = resp.GetWriteRequestPerHour()
	ns.NrObjects = resp.GetNrObjects()
	return ns, nil
}

// ListObjectKeyIterator implements datastor.Client.ListObjectKeyIterator
func (c *Client) ListObjectKeyIterator(ctx context.Context) (<-chan datastor.ObjectKeyResult, error) {
	// ensure a context is given
	if ctx == nil {
		panic("no context given")
	}

	group, ctx := errgroup.WithContext(ctx)
	ctx = c.contextWithMetadata(ctx)

	// create stream
	stream, err := c.objService.ListObjectKeys(ctx,
		&pb.ListObjectKeysRequest{})
	if err != nil {
		return nil, toContextErr(ctx, err)
	}

	// create output channel and start fetching from the stream
	ch := make(chan datastor.ObjectKeyResult, 1)
	group.Go(func() error {
		// fetch all objects possible
		var (
			input *pb.ListObjectKeysResponse
		)
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			// receive the next object, and check error as a first task to do
			input, err = stream.Recv()
			if err != nil {
				if err == io.EOF {
					// stream is done
					return nil
				}
				err = toContextErr(ctx, err)

				// an unexpected error has happened, exit with an error
				log.Errorf(
					"an error was received while receiving the key of an object for: %v",
					err)
				select {
				case ch <- datastor.ObjectKeyResult{Error: err}:
				case <-ctx.Done():
				}
				return err
			}

			// create the error/valid data result
			result := datastor.ObjectKeyResult{Key: input.GetKey()}
			if len(result.Key) == 0 {
				result.Error = datastor.ErrMissingKey
			}

			// return the result for the given key
			select {
			case ch <- result:
			case <-ctx.Done():
				return result.Error
			}
			if result.Error != nil {
				// if the result was an error, return also the error
				return result.Error
			}
		}
	})

	// launch the err group routine,
	// to close the output ch
	go func() {
		defer close(ch)
		err := group.Wait()
		if err != nil {
			log.Errorf(
				"ExistObjectIterator job group has exited with an error: %v",
				err)
		}
	}()

	return ch, nil
}

// SetReferenceList implements datastor.Client.SetReferenceList
func (c *Client) SetReferenceList(key []byte, refList []string) error {
	_, err := c.objService.SetReferenceList(
		c.contextWithMetadata(nil),
		&pb.SetReferenceListRequest{Key: key, ReferenceList: refList})
	if err != nil {
		return toErr(err)
	}
	return nil
}

// GetReferenceList implements datastor.Client.GetReferenceList
func (c *Client) GetReferenceList(key []byte) ([]string, error) {
	resp, err := c.objService.GetReferenceList(
		c.contextWithMetadata(nil), &pb.GetReferenceListRequest{Key: key})
	if err != nil {
		return nil, toErr(err)
	}
	refList := resp.GetReferenceList()
	if len(refList) == 0 {
		return nil, datastor.ErrMissingRefList
	}
	return refList, nil
}

// GetReferenceCount implements datastor.Client.GetReferenceCount
func (c *Client) GetReferenceCount(key []byte) (int64, error) {
	resp, err := c.objService.GetReferenceCount(
		c.contextWithMetadata(nil), &pb.GetReferenceCountRequest{Key: key})
	if err != nil {
		return 0, toErr(err)
	}
	return resp.GetCount(), nil
}

// AppendToReferenceList implements datastor.Client.AppendToReferenceList
func (c *Client) AppendToReferenceList(key []byte, refList []string) error {
	_, err := c.objService.AppendToReferenceList(
		c.contextWithMetadata(nil),
		&pb.AppendToReferenceListRequest{Key: key, ReferenceList: refList})
	if err != nil {
		return toErr(err)
	}
	return nil
}

// DeleteFromReferenceList implements datastor.Client.DeleteFromReferenceList
func (c *Client) DeleteFromReferenceList(key []byte, refList []string) (int64, error) {
	resp, err := c.objService.DeleteFromReferenceList(
		c.contextWithMetadata(nil),
		&pb.DeleteFromReferenceListRequest{Key: key, ReferenceList: refList})
	if err != nil {
		return 0, toErr(err)
	}
	return resp.GetCount(), nil
}

// DeleteReferenceList implements datastor.Client.DeleteReferenceList
func (c *Client) DeleteReferenceList(key []byte) error {
	_, err := c.objService.DeleteReferenceList(
		c.contextWithMetadata(nil), &pb.DeleteReferenceListRequest{Key: key})
	if err != nil {
		return toErr(err)
	}
	return nil
}

// Close implements datastor.Client.Close
func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) contextWithMetadata(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	var md metadata.MD
	if c.jwtTokenDefined {
		md = metadata.Pairs(
			rpctypes.MetaAuthKey, c.jwtToken,
			rpctypes.MetaLabelKey, c.label)
	} else {
		md = metadata.Pairs(rpctypes.MetaLabelKey, c.label)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

func toErr(err error) error {
	err = rpctypes.Error(err)
	if _, ok := err.(rpctypes.ZStorError); ok {
		if err, ok := expectedErrorMapping[err]; ok {
			return err
		}
		return err
	}
	return err
}

var expectedErrorMapping = map[error]error{
	rpctypes.ErrKeyNotFound:            datastor.ErrKeyNotFound,
	rpctypes.ErrObjectDataCorrupted:    datastor.ErrObjectDataCorrupted,
	rpctypes.ErrObjectRefListCorrupted: datastor.ErrObjectRefListCorrupted,
	rpctypes.ErrPermissionDenied:       datastor.ErrPermissionDenied,
}

func toContextErr(ctx context.Context, err error) error {
	if err := toErr(err); err != nil {
		return err
	}
	code := grpc.Code(err)
	switch code {
	case codes.DeadlineExceeded, codes.Canceled:
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	case codes.FailedPrecondition:
		err = grpc.ErrClientConnClosing
	}
	return err
}

// convertStatus converts pb.ObjectStatus datastor.ObjectStatus
func convertStatus(status pb.ObjectStatus) (datastor.ObjectStatus, error) {
	s, ok := _ProtoObjectStatusMapping[status]
	if !ok {
		log.Debugf("invalid (proto) object status %d received", status)
		return datastor.ObjectStatus(0), datastor.ErrInvalidStatus
	}
	return s, nil
}

var _ProtoObjectStatusMapping = map[pb.ObjectStatus]datastor.ObjectStatus{
	pb.ObjectStatusOK:        datastor.ObjectStatusOK,
	pb.ObjectStatusMissing:   datastor.ObjectStatusMissing,
	pb.ObjectStatusCorrupted: datastor.ObjectStatusCorrupted,
}
