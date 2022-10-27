package psr3

import (
	psr3v1 "go.buf.build/protocolbuffers/go/roadrunner-server/api/proto/logger/psr3/v1"
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

func (r *RPC) Emergency(in *psr3v1.Message, out *bool) error {
	r.log.Error(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Alert(in *psr3v1.Message, out *bool) error {
	r.log.Error(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Critical(in *psr3v1.Message, out *bool) error {
	r.log.Error(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Error(in *psr3v1.Message, out *bool) error {
	r.log.Error(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Warning(in *psr3v1.Message, out *bool) error {
	r.log.Warn(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Notice(in *psr3v1.Message, out *bool) error {
	r.log.Info(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Info(in *psr3v1.Message, out *bool) error {
	r.log.Info(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Debug(in *psr3v1.Message, out *bool) error {
	r.log.Debug(in.GetMessage(), zap.String("context", in.GetContext()))
	*out = true

	return nil
}

func (r *RPC) Log(_ *psr3v1.Message, _ *bool) error {
	return nil
}
