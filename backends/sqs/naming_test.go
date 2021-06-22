package sqs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipUrl(t *testing.T) {
	q1 := "https://sqs.us-east-2.amazonaws.com/123456789012/prod-foo"
	assert.True(t, SkipUrl(q1))
	q2 := "https://sqs.us-east-2.amazonaws.com/123456789012/prod_foo"
	assert.False(t, SkipUrl(q2))
}
