package app

import (
	"bytes"
	"context"
	stderrors "errors"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	apploggerV2 "github.com/roadrunner-server/api-go/v6/applogger/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errWriter always returns its preset error from Write — used to exercise
// the io.WriteString failure path in Log/LogWithContext.
type errWriter struct{ err error }

func (w *errWriter) Write(_ []byte) (int, error) { return 0, w.err }

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

func TestFormatRaw(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		args []*apploggerV2.LogAttrs
		want string
	}{
		{
			name: "nil args",
			msg:  "hello",
			args: nil,
			want: "hello\n",
		},
		{
			name: "empty args",
			msg:  "hello",
			args: []*apploggerV2.LogAttrs{},
			want: "hello\n",
		},
		{
			name: "no args already newline-terminated",
			msg:  "hello\n",
			args: nil,
			want: "hello\n",
		},
		{
			name: "single attr",
			msg:  "hello",
			args: []*apploggerV2.LogAttrs{{Key: "k1", Value: "v1"}},
			want: "hello k1:v1\n",
		},
		{
			name: "multiple attrs",
			msg:  "msg",
			args: []*apploggerV2.LogAttrs{
				{Key: "k1", Value: "v1"},
				{Key: "k2", Value: "v2"},
			},
			want: "msg k1:v1,k2:v2\n",
		},
		{
			name: "special chars in values",
			msg:  "msg",
			args: []*apploggerV2.LogAttrs{
				{Key: "url", Value: "http://example.com:8080"},
				{Key: "list", Value: "a,b,c"},
			},
			want: "msg url:http://example.com:8080,list:a,b,c\n",
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
		method func(r *service, ctx context.Context, msg string) error
		level  slog.Level
	}{
		{"Error", func(r *service, ctx context.Context, msg string) error {
			_, err := r.Error(ctx, connect.NewRequest(&apploggerV2.LogMessage{Message: msg}))
			return err
		}, slog.LevelError},
		{"Info", func(r *service, ctx context.Context, msg string) error {
			_, err := r.Info(ctx, connect.NewRequest(&apploggerV2.LogMessage{Message: msg}))
			return err
		}, slog.LevelInfo},
		{"Warning", func(r *service, ctx context.Context, msg string) error {
			_, err := r.Warning(ctx, connect.NewRequest(&apploggerV2.LogMessage{Message: msg}))
			return err
		}, slog.LevelWarn},
		{"Debug", func(r *service, ctx context.Context, msg string) error {
			_, err := r.Debug(ctx, connect.NewRequest(&apploggerV2.LogMessage{Message: msg}))
			return err
		}, slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &captureHandler{}
			s := &service{log: slog.New(h)}

			err := tt.method(s, t.Context(), "test message")
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
		method func(r *service, ctx context.Context, in *apploggerV2.LogEntry) error
		level  slog.Level
	}{
		{"ErrorWithContext", func(r *service, ctx context.Context, in *apploggerV2.LogEntry) error {
			_, err := r.ErrorWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelError},
		{"InfoWithContext", func(r *service, ctx context.Context, in *apploggerV2.LogEntry) error {
			_, err := r.InfoWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelInfo},
		{"WarningWithContext", func(r *service, ctx context.Context, in *apploggerV2.LogEntry) error {
			_, err := r.WarningWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelWarn},
		{"DebugWithContext", func(r *service, ctx context.Context, in *apploggerV2.LogEntry) error {
			_, err := r.DebugWithContext(ctx, connect.NewRequest(in))
			return err
		}, slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &captureHandler{}
			s := &service{log: slog.New(h)}

			entry := &apploggerV2.LogEntry{
				Message:  "ctx message",
				LogAttrs: []*apploggerV2.LogAttrs{{Key: "component", Value: "test"}},
			}

			err := tt.method(s, t.Context(), entry)
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
	s := &service{log: slog.New(h)}

	entry := &apploggerV2.LogEntry{
		Message: "multi attrs",
		LogAttrs: []*apploggerV2.LogAttrs{
			{Key: "k1", Value: "v1"},
			{Key: "k2", Value: "v2"},
			{Key: "k3", Value: "v3"},
		},
	}

	_, err := s.InfoWithContext(t.Context(), connect.NewRequest(entry))
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
	var buf bytes.Buffer
	s := &service{log: slog.New(slog.DiscardHandler), stderr: &buf}

	_, err := s.Log(t.Context(), connect.NewRequest(&apploggerV2.LogMessage{Message: "hello stderr\n"}))
	require.NoError(t, err)

	assert.Equal(t, "hello stderr\n", buf.String())
}

func TestRPCLogWithContext(t *testing.T) {
	var buf bytes.Buffer
	s := &service{log: slog.New(slog.DiscardHandler), stderr: &buf}

	entry := &apploggerV2.LogEntry{
		Message:  "hello",
		LogAttrs: []*apploggerV2.LogAttrs{{Key: "k", Value: "v"}},
	}
	_, err := s.LogWithContext(t.Context(), connect.NewRequest(entry))
	require.NoError(t, err)

	assert.Equal(t, "hello k:v\n", buf.String())
}

func TestRPCLogWriteFailureMapsToCodeInternal(t *testing.T) {
	want := stderrors.New("write failure")
	s := &service{log: slog.New(slog.DiscardHandler), stderr: &errWriter{err: want}}

	t.Run("Log", func(t *testing.T) {
		resp, err := s.Log(t.Context(), connect.NewRequest(&apploggerV2.LogMessage{Message: "x"}))
		require.Nil(t, resp)
		require.Error(t, err)
		assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		assert.ErrorIs(t, err, want)
	})

	t.Run("LogWithContext", func(t *testing.T) {
		resp, err := s.LogWithContext(t.Context(), connect.NewRequest(&apploggerV2.LogEntry{Message: "x"}))
		require.Nil(t, resp)
		require.Error(t, err)
		assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		assert.ErrorIs(t, err, want)
	})
}
