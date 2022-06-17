package config

import (
	"log"
	"time"

	"github.com/jessevdk/go-flags"
)

const (
	SQS    string = "sqs"
	Pubsub string = "pubsub"
)

type Config struct {
	Backend           string        `long:"backend" description:"Backend queueing system to use" required:"true"`
	Region            string        `long:"region" description:"Region to connect to (if applicable)"`
	MetricsMode       bool          `long:"metricsmode" description:"Start qproxy in metrics mode, to collect queued/inflight metrics per queue"`
	MetricsNamespace  string        `long:"metricsnamespace" description:"What namespace to collect additional metrics under" default:"prod"`
	MaxIdleConns      int           `long:"maxidleconns" description:"Maximum number of connections to hold to the backend" default:"1000"`
	MaxConnsPerHost   int           `long:"maxconnsperhost" description:"Maximum number of connections to have open to the backend" default:"10000"`
	DefaultRPCTimeout time.Duration `long:"defaultrpctimeout" description:"Default rpc timeout if none specified" default:"30s"`

	GRPCPort int `long:"grpcport" description:"Port for grpc server to listen on" default:"8887"`
	HTTPPort int `long:"httpport" description:"Port for http server to listen on" default:"8888"`

	WriteTimeout time.Duration `long:"writetimeout" description:"HTTP server write timeout" default:"0"`
	ReadTimeout  time.Duration `long:"readtimeout" description:"HTTP server read timeout" default:"0"`
	IdleTimeout  time.Duration `long:"idletimeout" description:"HTTP server idle timeout" default:"0"`

	TermSleep time.Duration `long:"termsleep" description:"How long to sleep before gracefully shutting down" default:"30s"`

	Profile    string `long:"profile" description:"Run a CPUProfile, output a file with this name"`
	MemProfile string `long:"memprofile" description:"Run a MemProfile, output a file with this name"`

	GrpcMaxConnectionAge      time.Duration `long:"grpc-max-connection-age" default:"300s" description:"duration a connection may exist before it will be closed by sending a GoAway."`
	GrpcMaxConnectionAgeGrace time.Duration `long:"grpc-max-connection-age-grace" default:"50s" description:"period after MaxConnectionAge after which the connection will be forcibly closed."`
}

func ParseConfig() *Config {
	var config Config
	parser := flags.NewParser(&config, flags.Default)
	if _, err := parser.Parse(); err != nil {
		log.Fatalf(err.Error())
	}
	return &config
}
