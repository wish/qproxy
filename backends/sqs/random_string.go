package sqs

import (
	"math/rand"
	"time"
)

const dedupIdLength = 128
const dedupIdCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"+
    "0123456789!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func creaeteMessageDedupId() string {
  b := make([]byte, dedupIdLength)
  for i := range b {
    b[i] = dedupIdCharset[seededRand.Intn(len(dedupIdCharset))]
  }
  return string(b)
}