package app

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/roadrunner-server/api-go/v6/applogger/v2/apploggerV2connect"
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

func (p *Plugin) RPC() (string, http.Handler) {
	return apploggerV2connect.NewAppLoggerServiceHandler(&service{log: p.log, stderr: os.Stderr})
}
