package app

import (
	"io"
	"os"
	"testing"

	v2 "github.com/roadrunner-server/api-go/v6/applogger/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// captureStderr redirects os.Stderr to a pipe for the duration of fn,
// returning whatever was written. Not parallel-safe.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stderr = w
	fn()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = old

	return string(out)
}

func TestFormatRaw(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		args []*v2.LogAttrs
		want string
	}{
		{
			name: "single attr",
			msg:  "hello",
			args: []*v2.LogAttrs{{Key: "k1", Value: "v1"}},
			want: "hello k1:v1",
		},
		{
			name: "multiple attrs",
			msg:  "msg",
			args: []*v2.LogAttrs{
				{Key: "k1", Value: "v1"},
				{Key: "k2", Value: "v2"},
			},
			want: "msg k1:v1,k2:v2",
		},
		{
			name: "special chars in values",
			msg:  "msg",
			args: []*v2.LogAttrs{
				{Key: "url", Value: "http://example.com:8080"},
				{Key: "list", Value: "a,b,c"},
			},
			want: "msg url:http://example.com:8080,list:a,b,c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRaw(tt.msg, tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCLogLevels(t *testing.T) {
	tests := []struct {
		name   string
		method func(r *RPC, msg string) error
		level  zapcore.Level
	}{
		{"Error", func(r *RPC, msg string) error { var b bool; return r.Error(msg, &b) }, zap.ErrorLevel},
		{"Info", func(r *RPC, msg string) error { var b bool; return r.Info(msg, &b) }, zap.InfoLevel},
		{"Warning", func(r *RPC, msg string) error { var b bool; return r.Warning(msg, &b) }, zap.WarnLevel},
		{"Debug", func(r *RPC, msg string) error { var b bool; return r.Debug(msg, &b) }, zap.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			rpc := &RPC{log: zap.New(core)}

			err := tt.method(rpc, "test message")
			require.NoError(t, err)

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, tt.level, entries[0].Level)
			assert.Equal(t, "test message", entries[0].Message)
		})
	}
}

func TestRPCWithContext(t *testing.T) {
	tests := []struct {
		name   string
		method func(r *RPC, in *v2.LogEntry) error
		level  zapcore.Level
	}{
		{"ErrorWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.ErrorWithContext(in, &resp) }, zap.ErrorLevel},
		{"InfoWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.InfoWithContext(in, &resp) }, zap.InfoLevel},
		{"WarningWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.WarningWithContext(in, &resp) }, zap.WarnLevel},
		{"DebugWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.DebugWithContext(in, &resp) }, zap.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			rpc := &RPC{log: zap.New(core)}

			entry := &v2.LogEntry{
				Message:  "ctx message",
				LogAttrs: []*v2.LogAttrs{{Key: "component", Value: "test"}},
			}

			err := tt.method(rpc, entry)
			require.NoError(t, err)

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, tt.level, entries[0].Level)
			assert.Equal(t, "ctx message", entries[0].Message)

			// Verify context field
			require.Len(t, entries[0].ContextMap(), 1)
			assert.Equal(t, "test", entries[0].ContextMap()["component"])
		})
	}
}

func TestRPCWithContextMultipleAttrs(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	rpc := &RPC{log: zap.New(core)}

	entry := &v2.LogEntry{
		Message: "multi attrs",
		LogAttrs: []*v2.LogAttrs{
			{Key: "k1", Value: "v1"},
			{Key: "k2", Value: "v2"},
			{Key: "k3", Value: "v3"},
		},
	}

	var resp v2.LogResponse
	err := rpc.InfoWithContext(entry, &resp)
	require.NoError(t, err)

	entries := logs.All()
	require.Len(t, entries, 1)

	ctx := entries[0].ContextMap()
	assert.Len(t, ctx, 3)
	assert.Equal(t, "v1", ctx["k1"])
	assert.Equal(t, "v2", ctx["k2"])
	assert.Equal(t, "v3", ctx["k3"])
}

func TestRPCLog(t *testing.T) {
	rpc := &RPC{log: zap.NewNop()}

	out := captureStderr(t, func() {
		var b bool
		err := rpc.Log("hello stderr\n", &b)
		require.NoError(t, err)
	})

	assert.Equal(t, "hello stderr\n", out)
}

func TestRPCLogWithContext(t *testing.T) {
	rpc := &RPC{log: zap.NewNop()}

	t.Run("empty attrs", func(t *testing.T) {
		out := captureStderr(t, func() {
			entry := &v2.LogEntry{Message: "no context"}
			var resp v2.LogResponse
			err := rpc.LogWithContext(entry, &resp)
			require.NoError(t, err)
		})
		assert.Equal(t, "no context", out)
	})

	t.Run("with attrs", func(t *testing.T) {
		out := captureStderr(t, func() {
			entry := &v2.LogEntry{
				Message:  "with context",
				LogAttrs: []*v2.LogAttrs{{Key: "src", Value: "worker"}},
			}
			var resp v2.LogResponse
			err := rpc.LogWithContext(entry, &resp)
			require.NoError(t, err)
		})
		assert.Equal(t, "with context src:worker", out)
	})
}
