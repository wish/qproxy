package sqs

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/aws/service/sqs"
	"github.com/kahuang/qproxy/rpc"
	"sync"
)

type Backend struct {
	nameMapping *sync.Map
	sqs         *sqs.SQS
}

func New(region string) (*SQS, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	cfg.Region = region

	svc := sqs.New(cfg)

	return &Backend{
		nameMapping: make(map[rpc.QueueId]string),
		sqs:         svc,
	}, nil
}

func (s *Backend) GetQueueUrl(ctx context.Context, in *rpc.QueueId) (string, error) {
	if url, ok := s.nameMapping.Load(rpc.QueueId{
		Namespace: in.Namespace,
		Name:      in.Name,
	}); ok {
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
		return nil, err
	}

	s.nameMapping.Store(rpc.QueueId{
		Namespace: in.Namespace,
		Name:      in.Name,
	}, resp.QueueUrl)
	return resp.QueueUrl, nil
}

func (s *Backend) ListQueues(ctx context.Context, in *rpc.ListQueuesRequest, stream rpc.QProxy_ListQueuesServer) (err error) {
	req = s.sqs.ListQueueRequest(&sqs.ListQueuesInput{
		QueueNamePrefix: in.Id.Namespace,
	})

	req.SetContext(ctx)

	// TODO: The ListQueues API has a 1000 object return limit, and no paging functionality
	// so we'll either need to do some extra work here to get _all_ results or add
	// a field to the response indicating that it was truncated
	resp, err = req.Send()
	if err != nil {
		return err
	}

	buf := make([]*QueueId, 0, 100)
	for idx, url := range resp.QueueUrls {
		if idx != 0 && idx%100 == 0 {
			stream.Send(&rpc.ListQueuesResponse{
				Queues: buf,
			})
			buf = make([]*QueueId, 0, 100)
		}

		if queueId, err := QueueUrlToQueueId(url); err != nil {
			return err
		} else if strings.Contains(queueId, in.Filter) {
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
			Queues: []*QueueId{},
		})
	}
	return nil
}

func (s *Backend) CreateQueue(ctx context.Context, in *rpc.CreateQueueRequest) (resp *rpc.CreateQueueResponse, err error) {
	req = s.sqs.CreateQueueRequest(&sqs.CreateQueueInput{
		QueueName:  QueueIdToName(in.Id),
		Attributes: in.Attributes,
	})

	req.SetContext(ctx)

	resp, err := req.send()
	if err != nil {
		return nil, err
	}

	return &rpc.CreateQueueResponse{}, nil
}

func (s *Backend) DeleteQueue(ctx context.Context, in *rpc.DeleteQueueRequest) (resp *rpc.DeleteQueueResponse, err error) {
	req = s.sqs.DeleteQueueRequest(&sqs.DeleteQueueInput{
		QueueName: QueueIdToName(in.Id),
	})

	req.SetContext(ctx)

	resp, err := req.send()
	if err != nil {
		return nil, err
	}

	return &rpc.DeleteQueueResponse{}, nil
}

func (s *Backend) ModifyQueue(ctx context.Context, in *rpc.ModifyQueueRequest) (resp *rpc.ModifyQueueResponse, err error) {
	req = s.sqs.SetQueueAttributesRequest(&sqs.SetQueueAttributesInput{
		QueueName:  QueueIdToName(in.Id),
		Attributes: in.Attributes,
	})

	req.SetContext(ctx)

	resp, err := req.send()
	if err != nil {
		return nil, err
	}

	return &rpc.ModifyQueueResponse{}, nil
}

func (s *Backend) PurgeQueue(ctx context.Context, in *rpc.PurgeQueueRequest) (resp *rpc.PurgeQueueResponse, err error) {
	req = s.sqs.PurgeQueueRequest(&sqs.PurgeQueueInput{
		QueueName: QueueIdToName(in.Id),
	})

	req.SetContext(ctx)

	resp, err := req.send()
	if err != nil {
		return nil, err
	}

	return &rpc.PurgeQueueResponse{}, nil
}

func (s *Backend) AckMessages(ctx context.Context, in *rpc.AckMessagesRequest) (resp *rpc.AckMessagesResponse, err error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	entries := make([]*s.sqs.DeleteMessageBatchRequestEntry, len(in.Receipts))
	for idx, receipt := range in.Receipts {
		entries = append(entries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            string(idx),
			ReceiptHandle: receipt.Id,
		})
	}

	req = s.sqs.DeleteMessageBatchRequest(&sqs.DeleteMessageBatchInput{
		QueueUrl: url,
		Entries:  entries,
	})

	req.SetContext(ctx)

	resp, err := req.send()
	if err != nil {
		return nil, err
	}

	failed := make([]*rpc.MessageReceipt, 0, len(in.Receipts))
	for idx, fail := range resp.Failed {
		failedReceipt := in.Receipts[int(idx)]
		failedReceipt.ErrorMessage = fail.Message
		failed = append(failed, failedReceipt)
	}
	return &rpc.AckMessagesResponse{Failed: failed}, nil
}

func (s *Backend) GetMessages(ctx context.Context, in *rpc.GetMessagesRequest) (resp *rpc.GetMessagesResponse, err error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	req = s.sqs.ReceiveMessageRequest(&ReceiveMessageInput{
		MessageAttributeNames: []string{"All"},
		QueueUrl:              url,
		WaitTimeSeconds:       in.LongPollSeconds,
		MaxNumberOfMessages:   in.MaxMessages,
		VisiblityTimeout:      in.AckDeadlineSeconds,
	})

	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}

	messages := make([]*rpc.Message, len(resp.Messages))
	for _, message := range resp.Messages {
		attributes := make(map[string]string)
		for key, valueStruct := range message.MessageAttributes {
			attributes[key] = valueStruct.StringValue
		}
		messages = append(messages, &Message{
			Data:       message.Body,
			Attributes: attributes,
			MessageReceipt: &rpc.MessageReceipt{
				Id: message.ReceiptHandle,
			},
		})
	}

	return &rpc.GetMessagesResponse{Messages: messages}, nil
}

func (s *Backend) PublishMessages(ctx context.Context, in *rpc.PublishMessagesRequest) (resp *rpc.PublishMessagesResponse, err error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	entries := make([]*s.sqs.SendMessageBatchRequestEntry, len(in.Messages))
	for idx, receipt := range in.Receipts {
		entries = append(entries, &sqs.DeleteMessageBatchRequestEntry{
			Id:                string(idx),
			MessageAttributes: in.Attributes,
			MessageBody:       in.Data,
		})
	}

	req = s.sqs.SendMessageBatchRequest(&sqs.SendMessageBatchInput{
		QueueUrl: url,
		Entries:  entries,
	})

	req.SetContext(ctx)

	resp, err := req.send()
	if err != nil {
		return nil, err
	}

	failed := make([]*rpc.FailedPublish, len(resp.Failed))
	for _, fail := range resp.Failed {
		failed = append(failed, &rpc.FailedPublish{
			Index:        int64(fail.Id),
			ErrorMessage: fail.Message,
		})
	}
	return &rpc.PublishMessagesResponse{Failed: failed}, nil
}

func (s *Backend) ModifyAckDeadline(ctx context.Context, in *rpc.ModifyAckDeadlineRequest) (resp *rpc.ModifyAckDeadlineResponse, err error) {
	url, err := s.GetQueueUrl(ctx, in.QueueId)
	if err != nil {
		return nil, err
	}

	req = s.sqs.ChangeMessageVisiblityRequest(&ChangeMessageVisibilityInput{
		QueueUrl:          url,
		ReceiptHandle:     in.Receipt.Id,
		VisibilityTimeout: in.AckDeadlineSeconds,
	})

	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}

	return &rpc.ModifyAckDeadlineResponse{}, nil
}
