package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	apploggerV1 "github.com/roadrunner-server/api-go/v6/applogger/v1"
)

// Subset of PSR-3 implemented over net/rpc (goridge). Each level has a
// message-only variant and a *WithContext variant that takes structured attrs.
// PSR-3 also defines emergency/alert/critical/notice — those are not exposed;
// producers can map them onto Error/Info as appropriate.
// https://github.com/php-fig/fig-standards/blob/master/accepted/PSR-3-logger-interface.md

type service struct {
	log    *slog.Logger
	stderr io.Writer // injectable so the error path in Log/LogWithContext is testable
}

func (r *service) Error(in string, _ *bool) error {
	r.log.ErrorContext(context.Background(), in)
	return nil
}

func (r *service) ErrorWithContext(in *apploggerV1.LogEntry, _ *apploggerV1.Response) error {
	r.log.LogAttrs(context.Background(), slog.LevelError, in.GetMessage(), buildAttrs(in.GetLogAttrs())...)
	return nil
}

func (r *service) Info(in string, _ *bool) error {
	r.log.InfoContext(context.Background(), in)
	return nil
}

func (r *service) InfoWithContext(in *apploggerV1.LogEntry, _ *apploggerV1.Response) error {
	r.log.LogAttrs(context.Background(), slog.LevelInfo, in.GetMessage(), buildAttrs(in.GetLogAttrs())...)
	return nil
}

func (r *service) Warning(in string, _ *bool) error {
	r.log.WarnContext(context.Background(), in)
	return nil
}

func (r *service) WarningWithContext(in *apploggerV1.LogEntry, _ *apploggerV1.Response) error {
	r.log.LogAttrs(context.Background(), slog.LevelWarn, in.GetMessage(), buildAttrs(in.GetLogAttrs())...)
	return nil
}

func (r *service) Debug(in string, _ *bool) error {
	r.log.DebugContext(context.Background(), in)
	return nil
}

func (r *service) DebugWithContext(in *apploggerV1.LogEntry, _ *apploggerV1.Response) error {
	r.log.LogAttrs(context.Background(), slog.LevelDebug, in.GetMessage(), buildAttrs(in.GetLogAttrs())...)
	return nil
}

func (r *service) Log(in string, _ *bool) error {
	if _, err := io.WriteString(r.stderr, ensureNewline(in)); err != nil {
		return fmt.Errorf("write log message to stderr: %w", err)
	}
	return nil
}

func (r *service) LogWithContext(in *apploggerV1.LogEntry, _ *apploggerV1.Response) error {
	if _, err := io.WriteString(r.stderr, formatRaw(in.GetMessage(), in.GetLogAttrs())); err != nil {
		return fmt.Errorf("write log entry to stderr: %w", err)
	}
	return nil
}

// formatRaw renders a log entry as a single plain-text line (terminated by a
// newline) for the raw stderr path, joining attrs as comma-separated key:value
// pairs.
func formatRaw(msg string, args []*apploggerV1.LogAttrs) string {
	if len(args) == 0 {
		return ensureNewline(msg)
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
	b.WriteByte('\n')
	return b.String()
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
func buildAttrs(args []*apploggerV1.LogAttrs) []slog.Attr {
	attrs := make([]slog.Attr, len(args))
	for i, a := range args {
		attrs[i] = slog.String(a.GetKey(), a.GetValue())
	}
	return attrs
}
