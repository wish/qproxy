package gateway

import (
	"io"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/wish/qproxy/rpc"
)

type QProxyDirectClient struct{ S rpc.QProxyServer }

func (m *QProxyDirectClient) ListQueues(ctx context.Context, in *rpc.ListQueuesRequest, opts ...grpc.CallOption) (rpc.QProxy_ListQueuesClient, error) {
	// TODO: buffer response channel?
	c := make(chan *rpc.ListQueuesResponse)
	errChan := make(chan error, 1)

	client := &LocalQueueIdStreamClient{ctx, c, errChan}
	server := &LocalQueueIdStreamClient{ctx, c, errChan}

	go func() {
		defer close(c)
		if err := m.S.ListQueues(in, server); err != nil {
			errChan <- err
		}
	}()

	return client, nil
}

func (m *QProxyDirectClient) GetQueue(ctx context.Context, in *rpc.GetQueueRequest, opts ...grpc.CallOption) (*rpc.GetQueueResponse, error) {
	return m.S.GetQueue(ctx, in)
}

func (m *QProxyDirectClient) CreateQueue(ctx context.Context, in *rpc.CreateQueueRequest, opts ...grpc.CallOption) (*rpc.CreateQueueResponse, error) {
	return m.S.CreateQueue(ctx, in)
}

func (m *QProxyDirectClient) DeleteQueue(ctx context.Context, in *rpc.DeleteQueueRequest, opts ...grpc.CallOption) (*rpc.DeleteQueueResponse, error) {
	return m.S.DeleteQueue(ctx, in)
}

func (m *QProxyDirectClient) ModifyQueue(ctx context.Context, in *rpc.ModifyQueueRequest, opts ...grpc.CallOption) (*rpc.ModifyQueueResponse, error) {
	return m.S.ModifyQueue(ctx, in)
}

func (m *QProxyDirectClient) PurgeQueue(ctx context.Context, in *rpc.PurgeQueueRequest, opts ...grpc.CallOption) (*rpc.PurgeQueueResponse, error) {
	return m.S.PurgeQueue(ctx, in)
}

func (m *QProxyDirectClient) AckMessages(ctx context.Context, in *rpc.AckMessagesRequest, opts ...grpc.CallOption) (*rpc.AckMessagesResponse, error) {
	return m.S.AckMessages(ctx, in)
}
func (m *QProxyDirectClient) GetMessages(ctx context.Context, in *rpc.GetMessagesRequest, opts ...grpc.CallOption) (*rpc.GetMessagesResponse, error) {
	return m.S.GetMessages(ctx, in)
}

func (m *QProxyDirectClient) PublishMessages(ctx context.Context, in *rpc.PublishMessagesRequest, opts ...grpc.CallOption) (*rpc.PublishMessagesResponse, error) {
	return m.S.PublishMessages(ctx, in)
}

func (m *QProxyDirectClient) ModifyAckDeadline(ctx context.Context, in *rpc.ModifyAckDeadlineRequest, opts ...grpc.CallOption) (*rpc.ModifyAckDeadlineResponse, error) {
	return m.S.ModifyAckDeadline(ctx, in)
}

// Control flow
func (m *QProxyDirectClient) Healthcheck(ctx context.Context, in *rpc.HealthcheckRequest, opts ...grpc.CallOption) (*rpc.HealthcheckResponse, error) {
	return m.S.Healthcheck(ctx, in)
}

// ListQueues local stream client
type LocalQueueIdStreamClient struct {
	ctx     context.Context
	c       chan *rpc.ListQueuesResponse
	errChan chan error
}

func (l *LocalQueueIdStreamClient) Context() context.Context {
	return l.ctx
}

func (l *LocalQueueIdStreamClient) Recv() (*rpc.ListQueuesResponse, error) {
	select {
	case r, ok := <-l.c:
		if !ok {
			return nil, io.EOF
		} else {
			return r, nil
		}
	case err := <-l.errChan:
		return nil, err

	case <-l.ctx.Done():
		return nil, l.ctx.Err()
	}
}

func (l *LocalQueueIdStreamClient) Send(r *rpc.ListQueuesResponse) error {
	select {
	case <-l.ctx.Done():
		return l.ctx.Err()
	case l.c <- r:
		return nil
	}
}

func (l *LocalQueueIdStreamClient) SendMsg(m interface{}) error  { return nil }
func (l *LocalQueueIdStreamClient) RecvMsg(m interface{}) error  { return nil }
func (l *LocalQueueIdStreamClient) SetHeader(metadata.MD) error  { return nil }
func (l *LocalQueueIdStreamClient) SendHeader(metadata.MD) error { return nil }
func (l *LocalQueueIdStreamClient) SetTrailer(metadata.MD)       {}

//ClientStream
func (l *LocalQueueIdStreamClient) Header() (metadata.MD, error) { return nil, nil }
func (l *LocalQueueIdStreamClient) Trailer() metadata.MD         { return nil }
func (l *LocalQueueIdStreamClient) CloseSend() error             { return nil }
