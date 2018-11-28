package qproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/jacksontj/dataman/metrics"
	"github.com/wish/qproxy/backends/sqs"
	"github.com/wish/qproxy/rpc"
)

type QProxyServer struct {
	config          *Config
	backend         rpc.QProxyServer
	metricsRegistry metrics.Registry
	m               QProxyMetrics
}

func NewServer(config *Config) (*QProxyServer, error) {
	registry := metrics.NewNamespaceRegistry("qproxy")
	m, err := NewQProxyMetrics(registry)
	if err != nil {
		return nil, err
	}

	server := QProxyServer{
		config:          config,
		metricsRegistry: registry,
		m:               m,
	}

	switch config.Backend {
	case SQS:
		backend, err := sqs.New(config.Region)
		if err != nil {
			return nil, err
		}
		server.backend = backend
	case Pubsub:
		return nil, fmt.Errorf("Pubsub not implemented yet")
	default:
		return nil, fmt.Errorf("No backend queueing system specified. Please specify a backend")
	}

	return &server, nil
}

func (s *QProxyServer) ListQueues(in *rpc.ListQueuesRequest, stream rpc.QProxy_ListQueuesServer) (err error) {
	start := time.Now()
	s.m.APIHits.WithValues("ListQueues", in.Namespace, "").Inc(1)
	defer func() {
		s.m.APILatency.WithValues("ListQueues", in.Namespace, "").Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithValues("ListQueues", in.Namespace, "").Inc(1)
		}
	}()
	return s.backend.ListQueues(in, stream)
}

func (s *QProxyServer) GetQueue(ctx context.Context, in *rpc.GetQueueRequest) (resp *rpc.GetQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("GetQueue", in.Id.Namespace, in.Id.Name).Inc(1)
	defer func() {
		s.m.APILatency.WithValues("GetQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithValues("GetQueue", in.Id.Namespace, in.Id.Name).Inc(1)
		}
	}()
	return s.backend.GetQueue(ctx, in)
}

func (s *QProxyServer) CreateQueue(ctx context.Context, in *rpc.CreateQueueRequest) (resp *rpc.CreateQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("CreateQueue", in.Id.Namespace, in.Id.Name).Inc(1)
	defer func() {
		s.m.APILatency.WithValues("CreateQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
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
		s.m.APILatency.WithValues("DeleteQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
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
		s.m.APILatency.WithValues("ModifyQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
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
		s.m.APILatency.WithValues("PurgeQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
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
		s.m.APILatency.WithValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		} else {
			s.m.Acknowledged.WithValues(in.QueueId.Namespace, in.QueueId.Name).Inc(uint64(len(in.Receipts) - len(resp.Failed)))
		}
	}()
	return s.backend.AckMessages(ctx, in)
}

func (s *QProxyServer) GetMessages(ctx context.Context, in *rpc.GetMessagesRequest) (resp *rpc.GetMessagesResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("GetMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
	defer func() {
		s.m.APILatency.WithValues("GetMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithValues("GetMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		} else {
			s.m.Received.WithValues(in.QueueId.Namespace, in.QueueId.Name).Inc(uint64(len(resp.Messages)))
		}
	}()
	return s.backend.GetMessages(ctx, in)
}

func (s *QProxyServer) PublishMessages(ctx context.Context, in *rpc.PublishMessagesRequest) (resp *rpc.PublishMessagesResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
	defer func() {
		s.m.APILatency.WithValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		} else {
			s.m.Published.WithValues(in.QueueId.Namespace, in.QueueId.Name).Inc(uint64(len(in.Messages) - len(resp.Failed)))
		}
	}()
	return s.backend.PublishMessages(ctx, in)
}

func (s *QProxyServer) ModifyAckDeadline(ctx context.Context, in *rpc.ModifyAckDeadlineRequest) (resp *rpc.ModifyAckDeadlineResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
	defer func() {
		s.m.APILatency.WithValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Inc(1)
		}
	}()
	return s.backend.ModifyAckDeadline(ctx, in)
}

func (s *QProxyServer) Healthcheck(ctx context.Context, in *rpc.HealthcheckRequest) (resp *rpc.HealthcheckResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithValues("Healthcheck", "", "").Inc(1)
	defer func() {
		s.m.APILatency.WithValues("Healthcheck", "", "").Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithValues("Healthcheck", "", "").Inc(1)
		}
	}()
	return s.backend.Healthcheck(ctx, in)
}
