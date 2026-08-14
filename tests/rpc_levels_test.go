package app_logger //nolint:stylecheck

import (
	"log/slog"
	"testing"

	"tests/helpers"

	applogger "github.com/roadrunner-server/app-logger/v6"
	rpcPlugin "github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppLoggerLevels(t *testing.T) {
	rr, _ := helpers.Start(t, levelsConfig, levelsRPCAddr,
		[]any{&rpcPlugin.Plugin{}, &applogger.Plugin{}})

	client := helpers.RPC(t, levelsRPCAddr)

	calls := []struct {
		method  string
		message string
		level   slog.Level
	}{
		{"app.Debug", "Debug message", slog.LevelDebug},
		{"app.Error", "Error message", slog.LevelError},
		{"app.Info", "Info message", slog.LevelInfo},
		{"app.Warning", "Warning message", slog.LevelWarn},
	}

	var ok bool
	for _, c := range calls {
		require.NoError(t, client.Call(c.method, c.message, &ok))
	}

	for _, c := range calls {
		assert.Equal(t, 1, rr.Logs.FilterLevelExact(c.level).FilterMessageSnippet(c.message).Len(),
			"%s should produce exactly one %s record", c.method, c.level)
	}
}

// TestAppLoggerRawLog covers app.Log, the only plain-string method that does not
// go through slog: it writes the message to the process stderr RoadRunner
// collects, so it must not show up in the logger sink.
func TestAppLoggerRawLog(t *testing.T) {
	rr, _ := helpers.Start(t, levelsConfig, levelsRPCAddr,
		[]any{&rpcPlugin.Plugin{}, &applogger.Plugin{}})

	client := helpers.RPC(t, levelsRPCAddr)

	const (
		rawMessage  = "raw stderr message"
		sinkMessage = "sink control message"
	)

	var ok bool
	require.NoError(t, client.Call("app.Log", rawMessage+"\n", &ok))
	// Calls on one client are served in order, so by the time app.Info answers a
	// record from app.Log would already be in the sink.
	require.NoError(t, client.Call("app.Info", sinkMessage, &ok))

	assert.Equal(t, 1, rr.Logs.FilterMessageSnippet(sinkMessage).Len(),
		"app.Info over the same connection should reach the logger sink")
	assert.Zero(t, rr.Logs.FilterMessageSnippet(rawMessage).Len(),
		"app.Log should write to the process stderr instead of the logger sink")
}
