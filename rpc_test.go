package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	v2 "github.com/roadrunner-server/api-go/v6/applogger/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureHandler records every record observed for assertions in tests.
type captureHandler struct {
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

// captureStderr redirects os.Stderr to a pipe for the duration of fn,
// returning whatever was written. Not parallel-safe.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)

	defer func() {
		_ = w.Close() // no-op if already closed below
		os.Stderr = old
		_ = r.Close()
	}()

	os.Stderr = w
	fn()
	_ = w.Close()
	out, _ := io.ReadAll(r)

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
			name: "nil args",
			msg:  "hello",
			args: nil,
			want: "hello",
		},
		{
			name: "empty args",
			msg:  "hello",
			args: []*v2.LogAttrs{},
			want: "hello",
		},
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
		level  slog.Level
	}{
		{"Error", func(r *RPC, msg string) error { var b bool; return r.Error(msg, &b) }, slog.LevelError},
		{"Info", func(r *RPC, msg string) error { var b bool; return r.Info(msg, &b) }, slog.LevelInfo},
		{"Warning", func(r *RPC, msg string) error { var b bool; return r.Warning(msg, &b) }, slog.LevelWarn},
		{"Debug", func(r *RPC, msg string) error { var b bool; return r.Debug(msg, &b) }, slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &captureHandler{}
			rpc := &RPC{log: slog.New(h)}

			err := tt.method(rpc, "test message")
			require.NoError(t, err)

			require.Len(t, h.records, 1)
			assert.Equal(t, tt.level, h.records[0].Level)
			assert.Equal(t, "test message", h.records[0].Message)
		})
	}
}

func TestRPCWithContext(t *testing.T) {
	tests := []struct {
		name   string
		method func(r *RPC, in *v2.LogEntry) error
		level  slog.Level
	}{
		{"ErrorWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.ErrorWithContext(in, &resp) }, slog.LevelError},
		{"InfoWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.InfoWithContext(in, &resp) }, slog.LevelInfo},
		{"WarningWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.WarningWithContext(in, &resp) }, slog.LevelWarn},
		{"DebugWithContext", func(r *RPC, in *v2.LogEntry) error { var resp v2.LogResponse; return r.DebugWithContext(in, &resp) }, slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &captureHandler{}
			rpc := &RPC{log: slog.New(h)}

			entry := &v2.LogEntry{
				Message:  "ctx message",
				LogAttrs: []*v2.LogAttrs{{Key: "component", Value: "test"}},
			}

			err := tt.method(rpc, entry)
			require.NoError(t, err)

			require.Len(t, h.records, 1)
			assert.Equal(t, tt.level, h.records[0].Level)
			assert.Equal(t, "ctx message", h.records[0].Message)

			attrs := collectAttrs(h.records[0])
			require.Len(t, attrs, 1)
			assert.Equal(t, "test", attrs["component"])
		})
	}
}

func TestRPCWithContextMultipleAttrs(t *testing.T) {
	h := &captureHandler{}
	rpc := &RPC{log: slog.New(h)}

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

	require.Len(t, h.records, 1)

	attrs := collectAttrs(h.records[0])
	assert.Len(t, attrs, 3)
	assert.Equal(t, "v1", attrs["k1"])
	assert.Equal(t, "v2", attrs["k2"])
	assert.Equal(t, "v3", attrs["k3"])
}

// collectAttrs walks a slog.Record's attributes into a flat map of string values.
func collectAttrs(r slog.Record) map[string]string {
	out := map[string]string{}
	r.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.String()
		return true
	})
	return out
}

func TestRPCLog(t *testing.T) {
	rpc := &RPC{log: slog.New(slog.DiscardHandler)}

	out := captureStderr(t, func() {
		var b bool
		err := rpc.Log("hello stderr\n", &b)
		require.NoError(t, err)
	})

	assert.Equal(t, "hello stderr\n", out)
}
