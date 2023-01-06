package app

import (
	"go.uber.org/zap"
)

const name = "app"

type Logger interface {
	NamedLogger(name string) *zap.Logger
}

type Plugin struct {
	log *zap.Logger
}

func (p *Plugin) Init(log Logger) error {
	p.log = log.NamedLogger(name)
	return nil
}

func (p *Plugin) Name() string {
	return name
}

func (p *Plugin) RPC() any {
	return &RPC{
		log: p.log,
	}
}
