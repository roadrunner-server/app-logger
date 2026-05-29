package app

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	apploggerV2 "github.com/roadrunner-server/api-go/v6/applogger/v2"
)

// Subset of PSR-3 implemented over Connect-RPC. Each level has a string-only
// variant and a *WithContext variant that takes structured attrs. PSR-3 also
// defines emergency/alert/critical/notice — those are not exposed; producers
// can map them onto Error/Info as appropriate.
// https://github.com/php-fig/fig-standards/blob/master/accepted/PSR-3-logger-interface.md

type service struct {
	log    *slog.Logger
	stderr io.Writer // injectable so the error path in Log/LogWithContext is testable
}

func (r *service) Error(ctx context.Context, req *connect.Request[apploggerV2.LogMessage]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.ErrorContext(ctx, req.Msg.GetMessage())
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) ErrorWithContext(ctx context.Context, req *connect.Request[apploggerV2.LogEntry]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.LogAttrs(ctx, slog.LevelError, req.Msg.GetMessage(), buildAttrs(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) Info(ctx context.Context, req *connect.Request[apploggerV2.LogMessage]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.InfoContext(ctx, req.Msg.GetMessage())
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) InfoWithContext(ctx context.Context, req *connect.Request[apploggerV2.LogEntry]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.LogAttrs(ctx, slog.LevelInfo, req.Msg.GetMessage(), buildAttrs(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) Warning(ctx context.Context, req *connect.Request[apploggerV2.LogMessage]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.WarnContext(ctx, req.Msg.GetMessage())
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) WarningWithContext(ctx context.Context, req *connect.Request[apploggerV2.LogEntry]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.LogAttrs(ctx, slog.LevelWarn, req.Msg.GetMessage(), buildAttrs(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) Debug(ctx context.Context, req *connect.Request[apploggerV2.LogMessage]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.DebugContext(ctx, req.Msg.GetMessage())
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) DebugWithContext(ctx context.Context, req *connect.Request[apploggerV2.LogEntry]) (*connect.Response[apploggerV2.LogResponse], error) {
	r.log.LogAttrs(ctx, slog.LevelDebug, req.Msg.GetMessage(), buildAttrs(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) Log(_ context.Context, req *connect.Request[apploggerV2.LogMessage]) (*connect.Response[apploggerV2.LogResponse], error) {
	if _, err := io.WriteString(r.stderr, ensureNewline(req.Msg.GetMessage())); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

func (r *service) LogWithContext(_ context.Context, req *connect.Request[apploggerV2.LogEntry]) (*connect.Response[apploggerV2.LogResponse], error) {
	if _, err := io.WriteString(r.stderr, formatRaw(req.Msg.GetMessage(), req.Msg.GetLogAttrs())); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apploggerV2.LogResponse{}), nil
}

// formatRaw renders a log entry as a single plain-text line (terminated by a
// newline) for the raw stderr path, joining attrs as comma-separated key:value
// pairs.
func formatRaw(msg string, args []*apploggerV2.LogAttrs) string {
	if len(args) == 0 {
		return ensureNewline(msg)
	}

	pairs := make([]string, len(args))
	for i, a := range args {
		pairs[i] = a.GetKey() + ":" + a.GetValue()
	}
	return msg + " " + strings.Join(pairs, ",") + "\n"
}

// ensureNewline returns s with exactly one trailing newline, leaving an
// already newline-terminated string unchanged so raw stderr writes neither run
// together nor gain a blank line.
func ensureNewline(s string) string {
	if len(s) == 0 || s[len(s)-1] != '\n' {
		return s + "\n"
	}
	return s
}

// buildAttrs converts protobuf LogAttrs into typed slog.Attr values,
// enabling LogAttrs calls that avoid the []any boxing overhead.
func buildAttrs(args []*apploggerV2.LogAttrs) []slog.Attr {
	attrs := make([]slog.Attr, len(args))
	for i, a := range args {
		attrs[i] = slog.String(a.GetKey(), a.GetValue())
	}
	return attrs
}
