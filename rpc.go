package app

import (
	"fmt"
	"io"
	"os"

	v1 "github.com/roadrunner-server/api/v4/build/applogger/v1"

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

func (r *RPC) Error2(in *v1.LogEntry, _ *v1.Response) error {
	r.log.Error(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Info(in string, _ *bool) error {
	r.log.Info(in)

	return nil
}

func (r *RPC) InfoWithContext(in *v1.LogEntry, _ *v1.Response) error {
	r.log.Info(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Warning(in string, _ *bool) error {
	r.log.Warn(in)

	return nil
}

func (r *RPC) WarningWithContext(in *v1.LogEntry, _ *v1.Response) error {
	r.log.Warn(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Debug(in string, _ *bool) error {
	r.log.Debug(in)

	return nil
}

func (r *RPC) DebugWithContext(in *v1.LogEntry, _ *v1.Response) error {
	r.log.Debug(in.GetMessage(), format(in.GetLogAttrs())...)

	return nil
}

func (r *RPC) Log(in string, _ *bool) error {
	_, err := io.WriteString(os.Stderr, in)
	if err != nil {
		return err
	}

	return nil
}

func (r *RPC) LogWithContext(in *v1.LogEntry, _ *v1.Response) error {
	// special case when we don't have any attributes
	if len(in.GetLogAttrs()) == 0 {
		_, err := io.WriteString(os.Stderr, in.GetMessage())
		if err != nil {
			return err
		}

		return nil
	}

	_, err := io.WriteString(os.Stderr, formatRaw(in.GetMessage(), in.GetLogAttrs()))
	if err != nil {
		return err
	}

	return nil
}

func formatRaw(msg string, args []*v1.LogAttrs) string {
	res := ""

	for i := 0; i < len(args); i++ {
		res += fmt.Sprintf("%s:%s,", args[i].GetKey(), args[i].GetValue())
	}

	return fmt.Sprintf("%s %s", msg, res[:len(res)-2]) // remove last comma
}

func format(args []*v1.LogAttrs) []zap.Field {
	fields := make([]zap.Field, 0, len(args))

	for _, v := range args {
		fields = append(fields, zap.String(v.GetValue(), v.GetValue()))
	}

	return fields
}
