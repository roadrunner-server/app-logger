package app

import (
	"io"
	"os"

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

func (r *RPC) Info(in string, _ *bool) error {
	r.log.Info(in)

	return nil
}

func (r *RPC) Warning(in string, _ *bool) error {
	r.log.Warn(in)

	return nil
}

func (r *RPC) Debug(in string, _ *bool) error {
	r.log.Debug(in)

	return nil
}

func (r *RPC) Log(in string, _ *bool) error {
	_, err := io.WriteString(os.Stderr, in)
	if err != nil {
		return err
	}

	return nil
}
