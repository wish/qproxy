package qproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/jacksontj/dataman/metrics"
	"github.com/kahuang/qproxy/backends/sqs"
	"github.com/kahuang/qproxy/rpc"
)

type QProxyServer struct {
	config          *Config
	backend         rpc.QProxyServer
	metricsRegistry metrics.Registry
	m               QProxyMetrics
}

func NewServer(config *Config) (*QproxyServer, err) {
	var backend rpc.QProxyServer
	switch config.Backend {
	case SQS:
		backend, err := sqs.New(config.Region)
		if err != nil {
			return nil, err
		}
	case Pubsub:
		return nil, fmt.Errorf("Pubsub not implemented yet")
	default:
		return nil, fmt.Errorf("No backend queueing system specified. Please specify a backend")
	}

	registry := metrics.NewNamespaceRegistry("qproxy")
	m, err := NewQProxyMetrics(registry)
	if err != nil {
		return nil, err
	}

	return QProxyServer{
		config:          config,
		backend:         backend,
		metricsRegistry: registry,
		m:               m,
	}, nil
}

func (s *QProxyServer) ListQueues(in *rpc.ListQueuesRequest, stream rpc.QProxy_ListQueuesServer) (err error) {
	start := time.Now()
	s.m.APIHits.WithValues("ListQueues", in.Namespace, "").Inc(1)
	defer func() {
		m.APILatency.WithValues("ListQueues", in.Namespace, "").Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("ListQueues", in.Namespace, "").Inc(1)
		}
	}()
	return s.backend.ListQueues(in, stream)
}

func (s *QProxyServer) CreateQueue(ctx context.Context, in *rpc.CreateQueueRequest) (resp *rpc.CreateQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("CreateQueue", in.Id.Namespace, in.Id.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("CreateQueue", in.Id.Namespace, in.Id.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("CreateQueue", in.Id.Namespace, in.Id.Name).Inc(1)
		}
	}()
	return s.backend.CreateQueue(ctx, in)
}

func (s *QProxyServer) DeleteQueue(ctx context.Context, in *rpc.DeleteQueueRequest) (resp *rpc.DeleteQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("DeleteQueue", in.Id.Namespace, in.Id.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("DeleteQueue", in.Id.Namespace, in.Id.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("DeleteQueue", in.Id.Namespace, in.Id.Name).Inc(1)
		}
	}()
	return s.backend.DeleteQueue(ctx, in)
}

func (s *QProxyServer) ModifyQueue(ctx context.Context, in *rpc.ModifyQueueRequest) (resp *rpc.ModifyQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("ModifyQueue", in.Id.Namespace, in.Id.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("ModifyQueue", in.Id.Namespace, in.Id.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("ModifyQueue", in.Id.Namespace, in.Id.Name).Inc(1)
		}
	}()
	return s.backend.ModifyQueue(ctx, in)
}

func (s *QProxyServer) PurgeQueue(ctx context.Context, in *rpc.PurgeQueueRequest) (resp *rpc.PurgeQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("PurgeQueue", in.Id.Namespace, in.Id.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("PurgeQueue", in.Id.Namespace, in.Id.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("PurgeQueue", in.Id.Namespace, in.Id.Name).Inc(1)
		}
	}()
	return s.backend.PurgeQueue(ctx, in)
}

func (s *QProxyServer) AckMessages(ctx context.Context, in *rpc.AckMessagesRequest) (resp *rpc.AckMessagesResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		} else {
			s.m.Acked.WithValues(in.QueueId.Namespace, in.QueueId.Name).Inc(len(in.Receipts) - len(resp.Failed))
		}
	}()
	return s.backend.AckMessage(ctx, in)
}

func (s *QProxyServer) GetMessages(ctx context.Context, in *rpc.GetMessagesRequest) (resp *rpc.GetMessagesResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("GetMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("GetMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("GetMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		} else {
			s.m.Received.WithValues(in.QueueId.Namespace, in.QueueId.Name).Inc(len(resp.Messages))
		}
	}()
	return s.backend.GetMessage(ctx, in)
}

func (s *QProxyServer) PublishMessages(ctx context.Context, in *rpc.PublishMessagesRequest) (resp *rpc.PublishMessagesResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		} else {
			s.m.Published.WithValues(in.QueueId.Namespace, in.QueueId.Name).Inc(len(in.Messages) - len(resp.Failed))
		}
	}()
	return s.backend.PublishMessages(ctx, in)
}

func (s *QProxyServer) ModifyAckDeadline(ctx context.Context, in *rpc.ModifyAckDeadlineRequest) (resp *rpc.ModifyAckDeadlineResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
	defer func() {
		m.APILatency.WithValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Observe(time.Now().Sub(start))
		if err != nil {
			s.m.APIErrors.WithValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		}
	}()
	return s.backend.ModifyAckDeadline(ctx, in)
}
