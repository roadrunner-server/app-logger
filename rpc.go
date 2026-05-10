package app

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	v2 "github.com/roadrunner-server/api-go/v6/applogger/v2"
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

func (r *service) Error(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Error(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) ErrorWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Error(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) Info(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Info(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) InfoWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Info(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) Warning(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Warn(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) WarningWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Warn(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) Debug(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Debug(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) DebugWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Debug(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) Log(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	if _, err := io.WriteString(r.stderr, req.Msg.GetMessage()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *service) LogWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	if _, err := io.WriteString(r.stderr, formatRaw(req.Msg.GetMessage(), req.Msg.GetLogAttrs())); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func formatRaw(msg string, args []*v2.LogAttrs) string {
	if len(args) == 0 {
		return msg
	}

	var b strings.Builder
	b.WriteString(msg)
	b.WriteByte(' ')
	for i, a := range args {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(a.GetKey())
		b.WriteByte(':')
		b.WriteString(a.GetValue())
	}
	return b.String()
}

func format(args []*v2.LogAttrs) []any {
	fields := make([]any, 0, len(args)*2)

	for _, v := range args {
		fields = append(fields, v.GetKey(), v.GetValue())
	}

	return fields
}
