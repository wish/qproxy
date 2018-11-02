package sqs

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/kahuang/qproxy/rpc"
)

type Backend struct {
	// TODO: LRU Cache?
	nameMapping *sync.Map
	sqs         *sqs.SQS

	// Perf optimization
	stringType *string
}

func New(region string) (*Backend, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	cfg.Region = region

	svc := sqs.New(cfg)

	stringType := "String"
	return &Backend{
		nameMapping: &sync.Map{},
		sqs:         svc,
		stringType:  &stringType,
	}, nil
}

func (s *Backend) GetQueueUrl(ctx context.Context, in *rpc.QueueId) (string, error) {
	if url, ok := s.nameMapping.Load(s.queueIdToKey(in)); ok {
		return url.(string), nil
	}

	// The mapping from QueueId -> QueueUrl does not exist in our cache.
	// Let's look it up
	req := s.sqs.GetQueueUrlRequest(&sqs.GetQueueUrlInput{
		QueueName: QueueIdToName(in),
	})

	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return "", err
	}
	s.updateNameMapping(in, resp.QueueUrl)

	return *resp.QueueUrl, nil
}

func (s *Backend) queueIdToKey(in *rpc.QueueId) string {
	return strings.Join([]string{in.Namespace, in.Name}, ",")
}

func (s *Backend) updateNameMapping(in *rpc.QueueId, url *string) {
	s.nameMapping.Store(s.queueIdToKey(in), *url)
}

func (s *Backend) ListQueues(in *rpc.ListQueuesRequest, stream rpc.QProxy_ListQueuesServer) (err error) {
	req := s.sqs.ListQueuesRequest(&sqs.ListQueuesInput{
		QueueNamePrefix: &in.Namespace,
	})

	ctx := stream.Context()
	req.SetContext(ctx)

	// TODO: The ListQueues API has a 1000 object return limit, and no paging functionality
	// so we'll either need to do some extra work here to get _all_ results or add
	// a field to the response indicating that it was truncated
	resp, err := req.Send()
	if err != nil {
		return err
	}

	buf := make([]*rpc.QueueId, 0, 100)
	for idx, url := range resp.QueueUrls {
		if idx != 0 && idx%100 == 0 {
			stream.Send(&rpc.ListQueuesResponse{
				Queues: buf,
			})
			buf = make([]*rpc.QueueId, 0, 100)
		}

		if queueId, err := QueueUrlToQueueId(url); err != nil {
			return err
		} else if strings.Contains(queueId.Name, in.Filter) {
			buf = append(buf, queueId)
		}
	}
	// Send any remaining queues not flushed
	stream.Send(&rpc.ListQueuesResponse{
		Queues: buf,
	})

	// Send a terminating trailer chunk if we haven't already
	if len(buf) > 0 {
		stream.Send(&rpc.ListQueuesResponse{
			Queues: []*rpc.QueueId{},
		})
	}
	return nil
}

func (s *Backend) CreateQueue(ctx context.Context, in *rpc.CreateQueueRequest) (*rpc.CreateQueueResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	req := s.sqs.CreateQueueRequest(&sqs.CreateQueueInput{
		QueueName:  &url,
		Attributes: in.Attributes,
	})

	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}
	s.updateNameMapping(in.Id, resp.QueueUrl)

	return &rpc.CreateQueueResponse{}, nil
}

func (s *Backend) DeleteQueue(ctx context.Context, in *rpc.DeleteQueueRequest) (*rpc.DeleteQueueResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	req := s.sqs.DeleteQueueRequest(&sqs.DeleteQueueInput{
		QueueUrl: &url,
	})

	req.SetContext(ctx)

	_, err = req.Send()
	if err != nil {
		return nil, err
	}

	return &rpc.DeleteQueueResponse{}, nil
}

func (s *Backend) ModifyQueue(ctx context.Context, in *rpc.ModifyQueueRequest) (*rpc.ModifyQueueResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	req := s.sqs.SetQueueAttributesRequest(&sqs.SetQueueAttributesInput{
		QueueUrl:   &url,
		Attributes: in.Attributes,
	})

	req.SetContext(ctx)

	_, err = req.Send()
	if err != nil {
		return nil, err
	}

	return &rpc.ModifyQueueResponse{}, nil
}

func (s *Backend) PurgeQueue(ctx context.Context, in *rpc.PurgeQueueRequest) (*rpc.PurgeQueueResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	req := s.sqs.PurgeQueueRequest(&sqs.PurgeQueueInput{
		QueueUrl: &url,
	})

	req.SetContext(ctx)

	_, err = req.Send()
	if err != nil {
		return nil, err
	}

	return &rpc.PurgeQueueResponse{}, nil
}

func (s *Backend) AckMessages(ctx context.Context, in *rpc.AckMessagesRequest) (*rpc.AckMessagesResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	entries := make([]sqs.DeleteMessageBatchRequestEntry, 0, len(in.Receipts))
	for idx, receipt := range in.Receipts {
		strIdx := strconv.Itoa(idx)
		entries = append(entries, sqs.DeleteMessageBatchRequestEntry{
			Id:            &strIdx,
			ReceiptHandle: &receipt.Id,
		})
	}

	req := s.sqs.DeleteMessageBatchRequest(&sqs.DeleteMessageBatchInput{
		QueueUrl: &url,
		Entries:  entries,
	})

	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}

	failed := make([]*rpc.MessageReceipt, 0, len(in.Receipts))
	for idx, fail := range resp.Failed {
		failedReceipt := in.Receipts[int(idx)]
		failedReceipt.ErrorMessage = *fail.Message
		failed = append(failed, failedReceipt)
	}
	return &rpc.AckMessagesResponse{Failed: failed}, nil
}

func (s *Backend) GetMessages(ctx context.Context, in *rpc.GetMessagesRequest) (*rpc.GetMessagesResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	req := s.sqs.ReceiveMessageRequest(&sqs.ReceiveMessageInput{
		MessageAttributeNames: []string{"All"},
		QueueUrl:              &url,
		WaitTimeSeconds:       &in.LongPollSeconds,
		MaxNumberOfMessages:   &in.MaxMessages,
		VisibilityTimeout:     &in.AckDeadlineSeconds,
	})

	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}

	messages := make([]*rpc.Message, 0, len(resp.Messages))
	for _, message := range resp.Messages {
		attributes := make(map[string]string)
		for key, valueStruct := range message.MessageAttributes {
			attributes[key] = *valueStruct.StringValue
		}
		messages = append(messages, &rpc.Message{
			Data:       *message.Body,
			Attributes: attributes,
			Receipt: &rpc.MessageReceipt{
				Id: *message.ReceiptHandle,
			},
		})
	}

	return &rpc.GetMessagesResponse{Messages: messages}, nil
}

func (s *Backend) PublishMessages(ctx context.Context, in *rpc.PublishMessagesRequest) (*rpc.PublishMessagesResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	entries := make([]sqs.SendMessageBatchRequestEntry, 0, len(in.Messages))
	for idx, message := range in.Messages {
		strIdx := strconv.Itoa(idx)
		attrs := make(map[string]sqs.MessageAttributeValue)
		for key, val := range message.Attributes {
			attrs[key] = sqs.MessageAttributeValue{
				DataType:    s.stringType,
				StringValue: &val,
			}
		}
		entries = append(entries, sqs.SendMessageBatchRequestEntry{
			Id:                &strIdx,
			MessageAttributes: attrs,
			MessageBody:       &message.Data,
		})
	}

	req := s.sqs.SendMessageBatchRequest(&sqs.SendMessageBatchInput{
		QueueUrl: &url,
		Entries:  entries,
	})

	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}

	failed := make([]*rpc.FailedPublish, 0, len(resp.Failed))
	for _, fail := range resp.Failed {
		if index, err := strconv.ParseInt(*fail.Id, 10, 64); err != nil {
			return nil, err
		} else {
			failed = append(failed, &rpc.FailedPublish{
				Index:        index,
				ErrorMessage: *fail.Message,
			})
		}
	}
	return &rpc.PublishMessagesResponse{Failed: failed}, nil
}

func (s *Backend) ModifyAckDeadline(ctx context.Context, in *rpc.ModifyAckDeadlineRequest) (*rpc.ModifyAckDeadlineResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	req := s.sqs.ChangeMessageVisibilityRequest(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          &url,
		ReceiptHandle:     &in.Receipt.Id,
		VisibilityTimeout: &in.AckDeadlineSeconds,
	})

	req.SetContext(ctx)

	_, err = req.Send()
	if err != nil {
		return nil, err
	}

	return &rpc.ModifyAckDeadlineResponse{}, nil
}

func (s *Backend) Healthcheck(ctx context.Context, in *rpc.HealthcheckRequest) (*rpc.HealthcheckResponse, error) {
	return &rpc.HealthcheckResponse{}, nil
}
