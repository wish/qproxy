package sqs

const sepChar = "_"
const forwardSlash = "/"

func QueueIdToName(id *rpc.QueueId) string {
	return strings.Join([]string{id.Namespace, id.Name}, sepChar)
}

func QueueUrlToQueueId(url string) (*rpc.QueueId, error) {
	// Example url: https://sqs.us-east-2.amazonaws.com/123456789012/MyQueue
	tokens := strings.Split(url, forwardSlash)
	name := tokens[len(tokens)-1]
	name_tokens := strings.SplitN(name, sepChar, 1)
	if len(name_tokens) != 2 {
		return nil, fmt.Errorf("Malformed queue name %v", name)
	}
	return &rpc.QueueId{
		Namespace: name_tokens[0],
		Name:      name_tokens[1],
	}, nil
}
