package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"connectrpc.com/connect"
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
		method func(r *RPC, ctx context.Context, msg string) error
		level  slog.Level
	}{
		{"Error", func(r *RPC, ctx context.Context, msg string) error {
			_, err := r.Error(ctx, connect.NewRequest(&v2.LogMessage{Message: msg}))
			return err
		}, slog.LevelError},
		{"Info", func(r *RPC, ctx context.Context, msg string) error {
			_, err := r.Info(ctx, connect.NewRequest(&v2.LogMessage{Message: msg}))
			return err
		}, slog.LevelInfo},
		{"Warning", func(r *RPC, ctx context.Context, msg string) error {
			_, err := r.Warning(ctx, connect.NewRequest(&v2.LogMessage{Message: msg}))
			return err
		}, slog.LevelWarn},
		{"Debug", func(r *RPC, ctx context.Context, msg string) error {
			_, err := r.Debug(ctx, connect.NewRequest(&v2.LogMessage{Message: msg}))
			return err
		}, slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &captureHandler{}
			rpc := &RPC{log: slog.New(h)}

			err := tt.method(rpc, t.Context(), "test message")
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
		method func(r *RPC, ctx context.Context, in *v2.LogEntry) error
		level  slog.Level
	}{
		{"ErrorWithContext", func(r *RPC, ctx context.Context, in *v2.LogEntry) error {
			_, err := r.ErrorWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelError},
		{"InfoWithContext", func(r *RPC, ctx context.Context, in *v2.LogEntry) error {
			_, err := r.InfoWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelInfo},
		{"WarningWithContext", func(r *RPC, ctx context.Context, in *v2.LogEntry) error {
			_, err := r.WarningWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelWarn},
		{"DebugWithContext", func(r *RPC, ctx context.Context, in *v2.LogEntry) error {
			_, err := r.DebugWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &captureHandler{}
			rpc := &RPC{log: slog.New(h)}

			entry := &v2.LogEntry{
				Message:  "ctx message",
				LogAttrs: []*v2.LogAttrs{{Key: "component", Value: "test"}},
			}

			err := tt.method(rpc, t.Context(), entry)
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

	_, err := rpc.InfoWithContext(t.Context(), connect.NewRequest(entry))
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
		_, err := rpc.Log(t.Context(), connect.NewRequest(&v2.LogMessage{Message: "hello stderr\n"}))
		require.NoError(t, err)
	})

	assert.Equal(t, "hello stderr\n", out)
}
