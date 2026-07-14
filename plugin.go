package app

import (
	"log/slog"
	"os"
)

const pluginName = "app"

type Logger interface {
	NamedLogger(name string) *slog.Logger
}

type Plugin struct {
	log *slog.Logger
}

func (p *Plugin) Init(log Logger) error {
	p.log = log.NamedLogger(pluginName)

	return nil
}

func (p *Plugin) Name() string {
	return pluginName
}

func (p *Plugin) RPC() any {
	return &service{log: p.log, stderr: os.Stderr}
}
