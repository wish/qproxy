package qproxy

import (
	"github.com/jessevdk/go-flags"
)

const (
	SQS    string = "sqs"
	Pubsub string = "pubsub"
)

type Config struct {
	Backend string `long:"backend" description:"Backend queueing system to use" required:"true"`
	Region  string `long:"region" description:"Region to connect to (if applicable)"`
}

func ParseConfig() *Config {
	var config Config
	parser := flags.NewParser(&config, flags.Default)
	if _, err := parser.Parse(); err != nil {
		log.Fatalf(err.Error())
	}
	return config
}
