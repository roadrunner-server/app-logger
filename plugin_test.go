package app

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLogger stands in for the logger plugin, recording the names it is asked
// for and handing back a logger writing into handler.
type fakeLogger struct {
	names   []string
	handler slog.Handler
}

func (f *fakeLogger) NamedLogger(name string) *slog.Logger {
	f.names = append(f.names, name)
	return slog.New(f.handler)
}

// The name Init asks for is the namespace records are tagged with, and the same
// constant that makes the RPC methods reachable as app.*.
func TestPluginInitNamesLogger(t *testing.T) {
	l := &fakeLogger{handler: &captureHandler{}}

	p := &Plugin{}
	require.NoError(t, p.Init(l))

	assert.Equal(t, []string{"app"}, l.names)
}

// The plugin name is the RPC service prefix, so PHP's Logger reaches the methods
// as app.Debug, app.Info and so on.
func TestPluginName(t *testing.T) {
	assert.Equal(t, "app", (&Plugin{}).Name())
}

func TestPluginRPCWiring(t *testing.T) {
	h := &captureHandler{}

	p := &Plugin{}
	require.NoError(t, p.Init(&fakeLogger{handler: h}))

	svc, ok := p.RPC().(*service)
	require.True(t, ok)

	// The raw path has to land on the process stderr RoadRunner collects.
	assert.Same(t, os.Stderr, svc.stderr)

	// The service logs through the logger Init was given, not a fresh one.
	require.NoError(t, svc.Info("probe", nil))
	require.Len(t, h.records, 1)
	assert.Equal(t, "probe", h.records[0].Message)
}
