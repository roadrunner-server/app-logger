package app

import (
	"go.uber.org/zap"
)

const pluginName = "app"

type Logger interface {
	NamedLogger(name string) *zap.Logger
}

type Plugin struct {
	log    *zap.Logger
	config *Config
}

type Configurer interface {
	// UnmarshalKey takes a single key and unmarshal it into a Struct.
	UnmarshalKey(name string, out any) error
	// Has checks if a config section exists.
	Has(name string) bool
	// RRVersion is the roadrunner current version
	RRVersion() string
}

func (p *Plugin) Init(cfg Configurer, log Logger) error {
	p.log = log.NamedLogger(pluginName)

	p.config = &Config{}
	err := cfg.UnmarshalKey(pluginName, &p.config)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) Name() string {
	return pluginName
}

func (p *Plugin) RPC() any {
	return &RPC{
		log: p.log,
		cfg: p.config,
	}
}
