package config

import (
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
	Topic              string
	ProducerAddr       string
	ConsumerAddr       string
	ConsumerChannel    string
	ConsumerHandlerNum int

	// etcd
	EtcdRegisteredAddr string
	EtcdEnable     bool
}

func InitConfig(file string) error {
	cfg := &Config{}
	if err := cfg.loadFromFile(file); err != nil {
		return err
	}
	if err := cfg.validate(); err != nil {
		return err
	}
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
	cfg.Topic = c.Section("queue").Key("topic").String()
	cfg.ProducerAddr = c.Section("queue").Key("producerAddr").String()
	cfg.ConsumerAddr = c.Section("queue").Key("consumerAddr").String()
	cfg.ConsumerChannel = c.Section("queue").Key("consumerChannel").String()
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
	cfg.EtcdEnable, _ = c.Section("etcd").Key("enable").Bool()

	return nil
}

func (cfg *Config) mergeDefaultSetting() {
	if cfg.LogLevel == 0 {
		cfg.LogLevel = 5
	}

	if len(cfg.ProducerAddr) == 0 {
		cfg.ProducerAddr = "127.0.0.1:4152"
	}
	if len(cfg.ConsumerAddr) == 0 {
		cfg.ConsumerAddr = "127.0.0.1:4152"
	}
	if cfg.ConsumerHandlerNum == 0 {
		cfg.ConsumerHandlerNum = 1
	}
	if len(cfg.Topic) == 0 {
		cfg.Topic = "wuzhc"
	}
	if len(cfg.ConsumerChannel) == 0 {
		cfg.ConsumerChannel = "go-chat"
	}

	if len(cfg.GrpcServerAddr) == 0 {
		cfg.GrpcServerAddr = "127.0.0.1:9002"
	}
	if len(cfg.GrpcGatewayAddr) == 0 {
		cfg.GrpcGatewayAddr = "127.0.0.1:8081"
	}

	if len(cfg.GinServerAddr) == 0 {
		cfg.GinServerAddr = "127.0.0.1:8080"
	}

	if len(cfg.EtcdRegisteredAddr) == 0 {
		cfg.EtcdRegisteredAddr = "127.0.0.1:2379"
	}
}

func (cfg *Config) validate() error {
	return nil
}
