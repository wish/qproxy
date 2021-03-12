package sqs

import (
	"fmt"
	"strings"

	"github.com/wish/qproxy/rpc"
)

const sepChar = "_"
const forwardSlash = "/"

func QueueIdToName(id *rpc.QueueId) *string {
	url := strings.Join([]string{id.Namespace, id.Name}, sepChar)
	if id.Type == rpc.QueueId_Fifo && (len(url) <= 5 || url[len(url)-5:] != ".fifo") {
		url = url + ".fifo"
	}
	return &url
}

func QueueUrlToQueueId(url string) (*rpc.QueueId, error) {
	// Example url: https://sqs.us-east-2.amazonaws.com/123456789012/MyQueue
	tokens := strings.Split(url, forwardSlash)
	name := tokens[len(tokens)-1]
	name_tokens := strings.SplitN(name, sepChar, 2)
	if len(name_tokens) != 2 {
		return nil, fmt.Errorf("Malformed queue name %v", name)
	}
	queueType := rpc.QueueId_Standard
	if name[len(name)-5:] == ".fifo" {
		queueType = rpc.QueueId_Fifo
	}
	return &rpc.QueueId{
		Namespace: name_tokens[0],
		Name:      name_tokens[1],
		Type:      queueType,
	}, nil
}
