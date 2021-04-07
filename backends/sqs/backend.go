package sqs

import (
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/wish/qproxy/config"
	"github.com/wish/qproxy/gateway"
	metrics "github.com/wish/qproxy/metrics"
	"github.com/wish/qproxy/rpc"
)

var SQSErrorCodes = []string{
	sqs.ErrCodeBatchEntryIdsNotDistinct,
	sqs.ErrCodeBatchRequestTooLong,
	sqs.ErrCodeEmptyBatchRequest,
	sqs.ErrCodeInvalidAttributeName,
	sqs.ErrCodeInvalidBatchEntryId,
	sqs.ErrCodeInvalidIdFormat,
	sqs.ErrCodeInvalidMessageContents,
	sqs.ErrCodeMessageNotInflight,
	sqs.ErrCodeOverLimit,
	sqs.ErrCodePurgeQueueInProgress,
	sqs.ErrCodeQueueDeletedRecently,
	sqs.ErrCodeQueueDoesNotExist,
	sqs.ErrCodeQueueNameExists,
	sqs.ErrCodeReceiptHandleIsInvalid,
	sqs.ErrCodeTooManyEntriesInBatchRequest,
	sqs.ErrCodeUnsupportedOperation,
}

type Backend struct {
	// TODO: LRU Cache?
	nameMapping *sync.Map
	sqs         *sqs.SQS

	m metrics.QProxyMetrics
	// Perf optimization
	stringType *string
}

func New(conf *config.Config, mets metrics.QProxyMetrics) (*Backend, error) {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        conf.MaxIdleConns,
			MaxIdleConnsPerHost: conf.MaxIdleConns,
			MaxConnsPerHost:     conf.MaxConnsPerHost,
		},
	}
	cfg := &aws.Config{
		Region:     aws.String(conf.Region),
		HTTPClient: client,
	}
	ses := session.Must(session.NewSession())

	svc := sqs.New(ses, cfg)

	stringType := "String"
	backend := Backend{
		nameMapping: &sync.Map{},
		sqs:         svc,
		m:           mets,
		stringType:  &stringType,
	}

	if conf.MetricsMode {
		go backend.collectMetrics(conf.MetricsNamespace)
	}
	return &backend, nil
}

func (s *Backend) collectMetrics(metricsNamespace string) {
	directClient := gateway.QProxyDirectClient{s}
	queues := make([]*rpc.QueueId, 0)
	collectTicker := time.NewTicker(15 * time.Second)
	updateTicker := time.NewTicker(5 * time.Minute)

	updateFunc := func() ([]*rpc.QueueId, error) {
		newQueues := make([]*rpc.QueueId, 0, 1000)
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		respClient, err := directClient.ListQueues(ctx, &rpc.ListQueuesRequest{
			Namespace: metricsNamespace,
		})
		if err != nil {
			return nil, err
		}
		for {
			results, err := respClient.Recv()
			if err == nil {
				if results == nil {
					break
				}
				newQueues = append(newQueues, results.Queues...)
			} else {
				if err == io.EOF {
					return newQueues, nil
				}
				return nil, err
			}
		}
		return newQueues, nil
	}

	collectFunc := func(id *rpc.QueueId, wg *sync.WaitGroup) {
		defer wg.Done()
		ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
		attrs, err := s.GetQueue(ctx, &rpc.GetQueueRequest{Id: id})
		if err == nil {
			true_name := QueueIdToName(id)
			queued, err := strconv.ParseInt(attrs.Attributes["ApproximateNumberOfMessages"], 10, 64)
			if err == nil {
				s.m.Queued.WithLabelValues(id.Namespace, id.Name, *true_name).Set(float64(queued))
			}
			inflight, err := strconv.ParseInt(attrs.Attributes["ApproximateNumberOfMessagesNotVisible"], 10, 64)
			if err == nil {
				s.m.Inflight.WithLabelValues(id.Namespace, id.Name, *true_name).Set(float64(inflight))
			}
		}
	}

	newQueues, err := updateFunc()
	if err == nil {
		queues = newQueues
	}

	for {
		select {
		case <-updateTicker.C:
			newQueues, err := updateFunc()
			// TOD: log if err
			if err == nil {
				queues = newQueues
			}
		case <-collectTicker.C:
			wg := sync.WaitGroup{}
			for _, queue := range queues {
        		wg.Add(1)
				go collectFunc(queue, &wg)
			}
			wg.Wait()
		}
	}
}

func (s *Backend) GetQueueUrl(ctx context.Context, in *rpc.QueueId) (string, error) {
	if url, ok := s.nameMapping.Load(QueueIdToName(in)); ok {
		return url.(string), nil
	}

	// The mapping from QueueId -> QueueUrl does not exist in our cache.
	// Let's look it up
	output, err := s.sqs.GetQueueUrlWithContext(ctx, &sqs.GetQueueUrlInput{
		QueueName: QueueIdToName(in),
	})

	if err != nil {
		return "", err
	}
	s.updateNameMapping(in, output.QueueUrl)

	return *output.QueueUrl, nil
}

func (s *Backend) updateNameMapping(in *rpc.QueueId, url *string) {
	s.nameMapping.Store(QueueIdToName(in), *url)
}

func (s *Backend) ListQueues(in *rpc.ListQueuesRequest, stream rpc.QProxy_ListQueuesServer) (err error) {
	buf := make([]*rpc.QueueId, 0, 100)
	input := &sqs.ListQueuesInput{
		MaxResults:      aws.Int64(1000),
		QueueNamePrefix: &in.Namespace,
	}
	ctx := stream.Context()
	var queues []*string
	err = s.sqs.ListQueuesPagesWithContext(ctx, input,
		func(page *sqs.ListQueuesOutput, lastPage bool) bool {
			queues = append(queues, page.QueueUrls...)
			return true
		})
	if err != nil {
		log.Printf("Got error while querying sqs: %v", err)
		return err
	}

	for idx, url := range queues {
		if idx != 0 && idx%100 == 0 {
			stream.Send(&rpc.ListQueuesResponse{
				Queues: buf,
			})
			buf = make([]*rpc.QueueId, 0, 100)
		}

		if queueId, err := QueueUrlToQueueId(*url); err != nil {
			log.Printf("Got error while converting queue url: %v", err)
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

func (s *Backend) GetQueue(ctx context.Context, in *rpc.GetQueueRequest) (*rpc.GetQueueResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	output, err := s.sqs.GetQueueAttributesWithContext(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       &url,
		AttributeNames: []*string{aws.String("All")},
	})

	if err != nil {
		return nil, err
	}

	attributes := make(map[string]string)
	for k, v := range output.Attributes {
		attributes[k] = *v
	}
	return &rpc.GetQueueResponse{Attributes: attributes}, nil
}

func (s *Backend) CreateQueue(ctx context.Context, in *rpc.CreateQueueRequest) (*rpc.CreateQueueResponse, error) {
	queueName := QueueIdToName(in.Id)
	attributes := make(map[string]*string)
	for k, v := range in.Attributes {
		value := v
		attributes[k] = &value
	}
	if in.Id.Type == rpc.QueueId_Fifo {
		fifoValue := "true"
		attributes["FifoQueue"] = &fifoValue
		dedupValue := "true"
		attributes["ContentBasedDeduplication"] = &dedupValue
	}
	output, err := s.sqs.CreateQueueWithContext(ctx, &sqs.CreateQueueInput{
		QueueName:  queueName,
		Attributes: attributes,
	})

	if err != nil {
		return nil, err
	}
	s.updateNameMapping(in.Id, output.QueueUrl)

	return &rpc.CreateQueueResponse{}, nil
}

func (s *Backend) DeleteQueue(ctx context.Context, in *rpc.DeleteQueueRequest) (*rpc.DeleteQueueResponse, error) {
	url, err := s.GetQueueUrl(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	_, err = s.sqs.DeleteQueueWithContext(ctx, &sqs.DeleteQueueInput{
		QueueUrl: &url,
	})

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

	attributes := make(map[string]*string)
	for k, v := range in.Attributes {
		value := v
		attributes[k] = &value
	}
	_, err = s.sqs.SetQueueAttributesWithContext(ctx, &sqs.SetQueueAttributesInput{
		QueueUrl:   &url,
		Attributes: attributes,
	})

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

	_, err = s.sqs.PurgeQueueWithContext(ctx, &sqs.PurgeQueueInput{
		QueueUrl: &url,
	})

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

	entries := make([]*sqs.DeleteMessageBatchRequestEntry, 0, len(in.Receipts))
	for idx, receipt := range in.Receipts {
		strIdx := strconv.Itoa(idx)
		handle := receipt.Id
		entries = append(entries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            &strIdx,
			ReceiptHandle: &handle,
		})
	}

	output, err := s.sqs.DeleteMessageBatchWithContext(ctx, &sqs.DeleteMessageBatchInput{
		QueueUrl: &url,
		Entries:  entries,
	})

	if err != nil {
		return nil, err
	}

	failed := make([]*rpc.MessageReceipt, 0, len(in.Receipts))
	for idx, fail := range output.Failed {
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

	output, err := s.sqs.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{aws.String("All")},
		QueueUrl:              &url,
		WaitTimeSeconds:       &in.LongPollSeconds,
		MaxNumberOfMessages:   &in.MaxMessages,
		VisibilityTimeout:     &in.AckDeadlineSeconds,
	})

	if err != nil {
		return nil, err
	}

	messages := make([]*rpc.Message, 0, len(output.Messages))
	for _, message := range output.Messages {
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

	entries := make([]*sqs.SendMessageBatchRequestEntry, 0, len(in.Messages))
	for idx, message := range in.Messages {
		strIdx := strconv.Itoa(idx)
		attrs := make(map[string]*sqs.MessageAttributeValue)
		for key, val := range message.Attributes {
			var pointerVal string
			pointerVal = val
			attrs[key] = &sqs.MessageAttributeValue{
				DataType:    s.stringType,
				StringValue: &pointerVal,
			}
		}
		entry := &sqs.SendMessageBatchRequestEntry{
			Id:                &strIdx,
			MessageAttributes: attrs,
			MessageBody:       &message.Data,
		}
		if in.QueueId.Type == rpc.QueueId_Fifo {
			value := "MessageGroup"
			entry.MessageGroupId = &value
		}
		entries = append(entries, entry)
	}

	output, err := s.sqs.SendMessageBatchWithContext(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: &url,
		Entries:  entries,
	})

	if err != nil {
		return nil, err
	}

	failed := make([]*rpc.FailedPublish, 0, len(output.Failed))
	for _, fail := range output.Failed {
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

	_, err = s.sqs.ChangeMessageVisibilityWithContext(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          &url,
		ReceiptHandle:     &in.Receipt.Id,
		VisibilityTimeout: &in.AckDeadlineSeconds,
	})

	if err != nil {
		return nil, err
	}

	return &rpc.ModifyAckDeadlineResponse{}, nil
}

func (s *Backend) Healthcheck(ctx context.Context, in *rpc.HealthcheckRequest) (*rpc.HealthcheckResponse, error) {
	return &rpc.HealthcheckResponse{}, nil
}
