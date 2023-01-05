package config

import (
	commonaccount "github.com/anytypeio/any-sync/accountservice"
	"github.com/anytypeio/any-sync/app"
	"github.com/anytypeio/any-sync/app/logger"
	"github.com/anytypeio/any-sync/commonspace"
	"github.com/anytypeio/any-sync/metric"
	"github.com/anytypeio/any-sync/net"
	"github.com/anytypeio/any-sync/nodeconf"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/node/storage"
	"gopkg.in/yaml.v3"
	"os"
)

const CName = "config"

func NewFromFile(path string) (c *Config, err error) {
	c = &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return
}

type Config struct {
	GrpcServer net.Config            `yaml:"grpcServer"`
	Account    commonaccount.Config  `yaml:"account"`
	APIServer  net.Config            `yaml:"apiServer"`
	Nodes      []nodeconf.NodeConfig `yaml:"nodes"`
	Space      commonspace.Config    `yaml:"space"`
	Storage    storage.Config        `yaml:"storage"`
	Metric     metric.Config         `yaml:"metric"`
	Log        logger.Config         `yaml:"log"`
}

func (c *Config) Init(a *app.App) (err error) {
	return
}

func (c Config) Name() (name string) {
	return CName
}

func (c Config) GetNet() net.Config {
	return c.GrpcServer
}

func (c Config) GetDebugNet() net.Config {
	return c.APIServer
}

func (c Config) GetAccount() commonaccount.Config {
	return c.Account
}

func (c Config) GetMetric() metric.Config {
	return c.Metric
}

func (c Config) GetSpace() commonspace.Config {
	return c.Space
}

func (c Config) GetStorage() storage.Config {
	return c.Storage
}

func (c Config) GetNodes() []nodeconf.NodeConfig {
	return c.Nodes
}
