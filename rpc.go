package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"connectrpc.com/connect"
	v2 "github.com/roadrunner-server/api-go/v6/applogger/v2"
)

/*
https://github.com/php-fig/fig-standards/blob/master/accepted/PSR-3-logger-interface.md
public function emergency($message, array $context = array());
public function alert($message, array $context = array());
public function critical($message, array $context = array());
public function error($message, array $context = array());
public function warning($message, array $context = array());
public function notice($message, array $context = array());
public function info($message, array $context = array());
public function debug($message, array $context = array());
public function log($level, $message, array $context = array());
*/

type RPC struct {
	log *slog.Logger
}

func (r *RPC) Error(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Error(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) ErrorWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Error(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) Info(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Info(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) InfoWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Info(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) Warning(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Warn(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) WarningWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Warn(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) Debug(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	r.log.Debug(req.Msg.GetMessage())
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) DebugWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	r.log.Debug(req.Msg.GetMessage(), format(req.Msg.GetLogAttrs())...)
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) Log(_ context.Context, req *connect.Request[v2.LogMessage]) (*connect.Response[v2.LogResponse], error) {
	if _, err := io.WriteString(os.Stderr, req.Msg.GetMessage()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v2.LogResponse{}), nil
}

func (r *RPC) LogWithContext(_ context.Context, req *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error) {
	if _, err := io.WriteString(os.Stderr, formatRaw(req.Msg.GetMessage(), req.Msg.GetLogAttrs())); err != nil {
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
