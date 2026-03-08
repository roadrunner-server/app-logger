package app

import (
	"fmt"
	"io"
	"os"

	v2 "github.com/roadrunner-server/api-go/v6/applogger/v2"

	"go.uber.org/zap"
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
	log *zap.Logger
}

func (r *RPC) Error(in string, _ *bool) error {
	r.log.Error(in)

	return nil
}

func (r *RPC) ErrorWithContext(in *v2.LogEntry, _ *v2.LogResponse) error {
	r.log.Error(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Info(in string, _ *bool) error {
	r.log.Info(in)

	return nil
}

func (r *RPC) InfoWithContext(in *v2.LogEntry, _ *v2.LogResponse) error {
	r.log.Info(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Warning(in string, _ *bool) error {
	r.log.Warn(in)

	return nil
}

func (r *RPC) WarningWithContext(in *v2.LogEntry, _ *v2.LogResponse) error {
	r.log.Warn(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Debug(in string, _ *bool) error {
	r.log.Debug(in)

	return nil
}

func (r *RPC) DebugWithContext(in *v2.LogEntry, _ *v2.LogResponse) error {
	r.log.Debug(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Log(in string, _ *bool) error {
	_, err := io.WriteString(os.Stderr, in)
	return err
}

func (r *RPC) LogWithContext(in *v2.LogEntry, _ *v2.LogResponse) error {
	// special case when we don't have any attributes
	if len(in.GetLogAttrs()) == 0 {
		_, err := io.WriteString(os.Stderr, in.GetMessage())
		return err
	}

	_, err := io.WriteString(os.Stderr, formatRaw(in.GetMessage(), in.GetLogAttrs()))
	return err
}

func formatRaw(msg string, args []*v2.LogAttrs) string {
	var buf []byte
	for _, a := range args {
		buf = fmt.Appendf(buf, "%s:%s,", a.GetKey(), a.GetValue())
	}

	return msg + " " + string(buf[:len(buf)-1])
}

func format(args []*v2.LogAttrs) []zap.Field {
	fields := make([]zap.Field, 0, len(args))

	for _, v := range args {
		fields = append(fields, zap.String(v.GetKey(), v.GetValue()))
	}

	return fields
}
