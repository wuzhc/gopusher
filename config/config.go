package config

import (
	"flag"
	"gopkg.in/ini.v1"
)

var Cfg *Config

type Config struct {
	cfg *ini.File

	BatchSendNum int // zero means send in order

	// log
	LogLevel        int
	LogReportCaller bool

	// gin
	GinServerAddr string

	// grpc
	GrpcServerAddr  string
	GrpcGatewayAddr string

	// queue
	QueueTopic         string
	QueueChannel       string
	ProducerAddr       string
	ConsumerAddr       string
	ConsumerHandlerNum int

	// etcd
	EtcdRegisteredAddr string
	EtcdEnable         bool
	EtcdListenKey      string
}

func InitConfig(file string) error {
	cfg := &Config{}
	if err := cfg.loadFromFile(file); err != nil {
		return err
	}
	if err := cfg.validate(); err != nil {
		return err
	}

	cfg.mergeFlagOptions()
	cfg.mergeDefaultSetting()

	Cfg = cfg
	return nil
}

func NewDefaultConfig() *Config {
	cfg := &Config{}
	cfg.mergeDefaultSetting()
	return cfg
}

func (cfg *Config) loadFromFile(file string) error {
	c, err := ini.Load(file)
	if err != nil {
		return err
	}

	cfg.cfg = c

	// nsq
	cfg.QueueTopic = c.Section("queue").Key("topic").String()
	cfg.QueueChannel = c.Section("queue").Key("channel").String()
	cfg.ProducerAddr = c.Section("queue").Key("producerAddr").String()
	cfg.ConsumerAddr = c.Section("queue").Key("consumerAddr").String()
	cfg.ConsumerHandlerNum, _ = c.Section("queue").Key("consumerHandlerNum").Int()

	// grpc
	cfg.GrpcServerAddr = c.Section("grpc").Key("serverAddr").String()
	cfg.GrpcGatewayAddr = c.Section("grpc").Key("gatewayAddr").String()

	// gin
	cfg.GinServerAddr = c.Section("gin").Key("serverAddr").String()

	// log
	cfg.LogLevel, _ = c.Section("log").Key("level").Int()
	cfg.LogReportCaller, _ = c.Section("log").Key("reportCaller").Bool()

	// etcd
	cfg.EtcdRegisteredAddr = c.Section("etcd").Key("registeredAddr").String()
	cfg.EtcdListenKey = c.Section("etcd").Key("listenKey").String()
	cfg.EtcdEnable, _ = c.Section("etcd").Key("enable").Bool()

	return nil
}

func (cfg *Config) mergeFlagOptions() {
	flag.StringVar(&cfg.GinServerAddr, "ginServerAddr", cfg.GinServerAddr, "the address of gin server")
	flag.StringVar(&cfg.GrpcServerAddr, "grpcServerAddr", cfg.GrpcServerAddr, "the address of grpc server")
	flag.StringVar(&cfg.GrpcGatewayAddr, "grpcGatewayAddr", cfg.GrpcGatewayAddr, "the address of grpc gateway server")
	flag.StringVar(&cfg.QueueChannel, "queueChannel", cfg.QueueChannel, "the channel for nsq consumer")
	flag.Parse()
}

func (cfg *Config) mergeDefaultSetting() {
	// log
	if cfg.LogLevel == 0 {
		cfg.LogLevel = 5
	}

	// queue
	if len(cfg.ProducerAddr) == 0 {
		cfg.ProducerAddr = "127.0.0.1:4152"
	}
	if len(cfg.ConsumerAddr) == 0 {
		cfg.ConsumerAddr = "127.0.0.1:4152"
	}
	if cfg.ConsumerHandlerNum == 0 {
		cfg.ConsumerHandlerNum = 1
	}
	if len(cfg.QueueTopic) == 0 {
		cfg.QueueTopic = "gopusher"
	}
	if len(cfg.QueueChannel) == 0 {
		cfg.QueueChannel = "one"
	}

	// gin
	if len(cfg.GinServerAddr) == 0 {
		cfg.GinServerAddr = "127.0.0.1:18080"
	}

	// grpc
	if len(cfg.GrpcServerAddr) == 0 {
		cfg.GrpcServerAddr = "127.0.0.1:18081"
	}
	if len(cfg.GrpcGatewayAddr) == 0 {
		cfg.GrpcGatewayAddr = "127.0.0.1:18082"
	}

	// etcd
	if len(cfg.EtcdRegisteredAddr) == 0 {
		cfg.EtcdRegisteredAddr = "127.0.0.1:2379"
	}
	if len(cfg.EtcdListenKey) == 0 {
		cfg.EtcdListenKey = "/gopuser"
	}
}

func (cfg *Config) validate() error {
	return nil
}
