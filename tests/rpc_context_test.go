package app_logger //nolint:stylecheck

import (
	"log/slog"
	"testing"

	"tests/helpers"

	apploggerV1 "github.com/roadrunner-server/api-go/v6/applogger/v1"
	applogger "github.com/roadrunner-server/app-logger/v6"
	rpcPlugin "github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppLoggerWithContext(t *testing.T) {
	rr, _ := helpers.Start(t, contextConfig, contextRPCAddr,
		[]any{&rpcPlugin.Plugin{}, &applogger.Plugin{}})

	client := helpers.RPC(t, contextRPCAddr)

	calls := []struct {
		method string
		entry  *apploggerV1.LogEntry
		level  slog.Level
	}{
		{
			method: "app.DebugWithContext",
			entry: &apploggerV1.LogEntry{Message: "Debug context message", LogAttrs: []*apploggerV1.LogAttrs{
				{Key: "component", Value: "test"},
			}},
			level: slog.LevelDebug,
		},
		{
			method: "app.ErrorWithContext",
			entry: &apploggerV1.LogEntry{Message: "Error context message", LogAttrs: []*apploggerV1.LogAttrs{
				{Key: "error_code", Value: "500"},
				{Key: "trace", Value: "stack_trace_here"},
			}},
			level: slog.LevelError,
		},
		{
			method: "app.InfoWithContext",
			entry: &apploggerV1.LogEntry{Message: "Info context message", LogAttrs: []*apploggerV1.LogAttrs{
				{Key: "request_id", Value: "12345"},
				{Key: "user", Value: "john"},
			}},
			level: slog.LevelInfo,
		},
		{
			method: "app.WarningWithContext",
			entry: &apploggerV1.LogEntry{Message: "Warning context message", LogAttrs: []*apploggerV1.LogAttrs{
				{Key: "threshold", Value: "90"},
			}},
			level: slog.LevelWarn,
		},
	}

	var resp apploggerV1.Response
	for _, c := range calls {
		require.NoError(t, client.Call(c.method, c.entry, &resp))
	}

	for _, c := range calls {
		assert.Equal(t, 1, rr.Logs.FilterLevelExact(c.level).FilterMessageSnippet(c.entry.GetMessage()).Len(),
			"%s should produce exactly one %s record", c.method, c.level)

		for _, a := range c.entry.GetLogAttrs() {
			assert.Equal(t, 1, rr.Logs.FilterAttr(a.GetKey(), a.GetValue()).Len(),
				"%s should carry attr %s", c.method, a.GetKey())
		}
	}
}

// TestAppLoggerRawLogWithContext covers app.LogWithContext, which renders the
// entry as a plain line on the process stderr instead of handing it to slog.
func TestAppLoggerRawLogWithContext(t *testing.T) {
	rr, _ := helpers.Start(t, contextConfig, contextRPCAddr,
		[]any{&rpcPlugin.Plugin{}, &applogger.Plugin{}})

	client := helpers.RPC(t, contextRPCAddr)

	const (
		rawMessage  = "raw stderr context message"
		sinkMessage = "sink control context message"
	)

	raw := &apploggerV1.LogEntry{Message: rawMessage, LogAttrs: []*apploggerV1.LogAttrs{
		{Key: "source", Value: "worker"},
	}}
	control := &apploggerV1.LogEntry{Message: sinkMessage, LogAttrs: []*apploggerV1.LogAttrs{
		{Key: "control", Value: "sink"},
	}}

	var resp apploggerV1.Response
	require.NoError(t, client.Call("app.LogWithContext", raw, &resp))
	// Calls on one client are served in order, so by the time InfoWithContext
	// answers a record from LogWithContext would already be in the sink.
	require.NoError(t, client.Call("app.InfoWithContext", control, &resp))

	assert.Equal(t, 1, rr.Logs.FilterLevelExact(slog.LevelInfo).FilterMessageSnippet(sinkMessage).Len(),
		"app.InfoWithContext over the same connection should reach the logger sink")
	assert.Equal(t, 1, rr.Logs.FilterAttr("control", "sink").Len(),
		"the control record should carry its attr")

	assert.Zero(t, rr.Logs.FilterMessageSnippet(rawMessage).Len(),
		"app.LogWithContext should write to the process stderr instead of the logger sink")
	assert.Zero(t, rr.Logs.FilterAttr("source", "worker").Len(),
		"the raw entry attrs should not reach the logger sink either")
}
