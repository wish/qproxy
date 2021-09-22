package qproxy

import (
	"context"
	"fmt"
	"log"
	"time"
	"strconv"

	"github.com/wish/qproxy/backends/sqs"
	"github.com/wish/qproxy/config"
	qmetrics "github.com/wish/qproxy/metrics"
	"github.com/wish/qproxy/rpc"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
)

type QProxyServer struct {
	config  *config.Config
	backend rpc.QProxyServer
	m       qmetrics.QProxyMetrics
}

func NewServer(conf *config.Config) (*QProxyServer, error) {
	m, err := qmetrics.NewQProxyMetrics()
	if err != nil {
		return nil, err
	}

	server := QProxyServer{
		config: conf,
		m:      m,
	}

	switch conf.Backend {
	case config.SQS:
		backend, err := sqs.New(conf, m)
		if err != nil {
			return nil, err
		}
		server.backend = backend
	case config.Pubsub:
		return nil, fmt.Errorf("Pubsub not implemented yet")
	default:
		return nil, fmt.Errorf("No backend queueing system specified. Please specify a backend")
	}

	return &server, nil
}

type ListQueuesServerStream struct {
	rpc.QProxy_ListQueuesServer
	NewCtx context.Context
}

func (x ListQueuesServerStream) Context() context.Context {
	return x.NewCtx
}

func (s *QProxyServer) setContextTimeout(ctx context.Context, timeout int64) context.Context {
	if timeout <= int64(0) {
		ctx, _ = context.WithTimeout(ctx, s.config.DefaultRPCTimeout)
	} else {
		ctx, _ = context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	}
	return ctx
}

func (s *QProxyServer) ListQueues(in *rpc.ListQueuesRequest, stream rpc.QProxy_ListQueuesServer) (err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("ListQueues", in.Namespace, "").Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("ListQueues", in.Namespace, "").Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("ListQueues", in.Namespace, "", s.ParseError(err)).Inc()
			log.Println("Error ListQueues: ", err)
		}
	}()
	streamWithTimeout := ListQueuesServerStream{
		QProxy_ListQueuesServer: stream,
		NewCtx:                  s.setContextTimeout(stream.Context(), in.RPCTimeout),
	}
	return s.backend.ListQueues(in, streamWithTimeout)
}

func (s *QProxyServer) GetQueue(ctx context.Context, in *rpc.GetQueueRequest) (resp *rpc.GetQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("GetQueue", in.Id.Namespace, in.Id.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("GetQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("GetQueue", in.Id.Namespace, in.Id.Name, s.ParseError(err)).Inc()
			log.Println("Error GetQueue: ", err)
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.GetQueue(ctx, in)
}

func (s *QProxyServer) CreateQueue(ctx context.Context, in *rpc.CreateQueueRequest) (resp *rpc.CreateQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("CreateQueue", in.Id.Namespace, in.Id.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("CreateQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("CreateQueue", in.Id.Namespace, in.Id.Name, s.ParseError(err)).Inc()
			log.Println("Error CreateQueue: ", err)
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.CreateQueue(ctx, in)
}

func (s *QProxyServer) DeleteQueue(ctx context.Context, in *rpc.DeleteQueueRequest) (resp *rpc.DeleteQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("DeleteQueue", in.Id.Namespace, in.Id.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("DeleteQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("DeleteQueue", in.Id.Namespace, in.Id.Name, s.ParseError(err)).Inc()
			log.Println("Error DeleteQueue: ", err)
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.DeleteQueue(ctx, in)
}

func (s *QProxyServer) ModifyQueue(ctx context.Context, in *rpc.ModifyQueueRequest) (resp *rpc.ModifyQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("ModifyQueue", in.Id.Namespace, in.Id.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("ModifyQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("ModifyQueue", in.Id.Namespace, in.Id.Name, s.ParseError(err)).Inc()
			log.Println("Error ModifyQueue: ", err)
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.ModifyQueue(ctx, in)
}

func (s *QProxyServer) PurgeQueue(ctx context.Context, in *rpc.PurgeQueueRequest) (resp *rpc.PurgeQueueResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("PurgeQueue", in.Id.Namespace, in.Id.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("PurgeQueue", in.Id.Namespace, in.Id.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("PurgeQueue", in.Id.Namespace, in.Id.Name, s.ParseError(err)).Inc()
			log.Println("Error PurgeQueue: ", err)
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.PurgeQueue(ctx, in)
}

func (s *QProxyServer) AckMessages(ctx context.Context, in *rpc.AckMessagesRequest) (resp *rpc.AckMessagesResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("AckMessages", in.QueueId.Namespace, in.QueueId.Name, s.ParseError(err)).Inc()
			log.Println("Error AckMessages: ", err)
		} else {
			s.m.Acknowledged.WithLabelValues(in.QueueId.Namespace, in.QueueId.Name,*sqs.QueueIdToName(in.QueueId)).Add(float64(len(in.Receipts) - len(resp.Failed)))
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.AckMessages(ctx, in)
}

func (s *QProxyServer) GetMessages(ctx context.Context, in *rpc.GetMessagesRequest) (resp *rpc.GetMessagesResponse, err error) {
	start := time.Now()
	apiName := "GetMessages"
	if in.LongPollSeconds <= 0 {
		apiName = "GetMessagesShortPoll"
	}
	s.m.APIHits.WithLabelValues(apiName, in.QueueId.Namespace, in.QueueId.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues(apiName, in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues(apiName, in.QueueId.Namespace, in.QueueId.Name, s.ParseError(err)).Inc()
			log.Println("Error GetMessages: ", err)
		} else {
			s.m.Received.WithLabelValues(in.QueueId.Namespace, in.QueueId.Name).Add(float64(len(resp.Messages)))
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.GetMessages(ctx, in)
}

func (s *QProxyServer) PublishMessages(ctx context.Context, in *rpc.PublishMessagesRequest) (resp *rpc.PublishMessagesResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("PublishMessages", in.QueueId.Namespace, in.QueueId.Name, s.ParseError(err)).Inc()
			log.Println("Error PublishMessages: ", err)
		} else {
			s.m.Published.WithLabelValues(in.QueueId.Namespace, in.QueueId.Name, *sqs.QueueIdToName(in.QueueId)).Add(float64(len(in.Messages) - len(resp.Failed)))
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.PublishMessages(ctx, in)
}

func (s *QProxyServer) ModifyAckDeadline(ctx context.Context, in *rpc.ModifyAckDeadlineRequest) (resp *rpc.ModifyAckDeadlineResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name).Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("ModifyAckDeadline", in.QueueId.Namespace, in.QueueId.Name, s.ParseError(err)).Inc()
			log.Println("Error ModifyAckDeadline: ", err)
		}
	}()
	ctx = s.setContextTimeout(ctx, in.RPCTimeout)
	return s.backend.ModifyAckDeadline(ctx, in)
}

func (s *QProxyServer) Healthcheck(ctx context.Context, in *rpc.HealthcheckRequest) (resp *rpc.HealthcheckResponse, err error) {
	start := time.Now()
	s.m.APIHits.WithLabelValues("Healthcheck", "", "").Inc()
	defer func() {
		s.m.APILatency.WithLabelValues("Healthcheck", "", "").Observe(float64(time.Now().Sub(start)))
		if err != nil {
			s.m.APIErrors.WithLabelValues("Healthcheck", "", "", s.ParseError(err)).Inc()
			log.Println("Error Healthcheck: ", err)
		}
	}()
	return s.backend.Healthcheck(ctx, in)
}

var RequestErrorCodes = []string {
    request.ErrCodeSerialization,
    request.ErrCodeRead,
    request.ErrCodeResponseTimeout,
    request.ErrCodeInvalidPresignExpire,
    request.CanceledErrorCode,
    request.ErrCodeRequestError,
    request.InvalidParameterErrCode,
    request.ParamRequiredErrCode,
    request.ParamMinValueErrCode,
    request.ParamMinLenErrCode,
    request.ParamMaxLenErrCode,
    request.ParamFormatErrCode,
    request.HandlerResponseTimeout,
    request.WaiterResourceNotReadyErrorCode,
}

func (s *QProxyServer) ParseError(err error) string {
    if awsErr, ok := err.(awserr.Error); ok {
        if awsErr.Code() == "MissingRegion" {
            return "MissingRegionError"
        }
        if awsErr.Code() == "RequestCanceled" {
            if origErr := awsErr.OrigErr(); origErr != nil {
                if origErr.Error() == "context canceled" {
                    return "RequestCanceledErrorContextCanceled"
                } else if origErr.Error() == "context deadline exceeded" {
                    return "RequestCanceledErrorDeadlineExceeded"
                }
            }
            return "RequestCanceledError"
        }
        for _, b := range sqs.SQSErrorCodes {
            if awsErr.Code() == b {
                return "SQSBackendError"
            }
        }
        for _, b := range RequestErrorCodes {
            if awsErr.Code() == b {
                return "AWSRequestError"
            }
        }
        if er, ok := err.(awserr.RequestFailure); ok {
            return "AWSRequestFailureStatus"+strconv.Itoa(er.StatusCode())
        }
        return "UnkownAWSError"
    }
    return "UnkownError"
}
