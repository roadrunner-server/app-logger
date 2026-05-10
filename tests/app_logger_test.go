package app_logger //nolint:stylecheck

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	mocklogger "tests/mock"

	"connectrpc.com/connect"
	v2 "github.com/roadrunner-server/api-go/v6/applogger/v2"
	"github.com/roadrunner-server/api-go/v6/applogger/v2/apploggerV2connect"
	applogger "github.com/roadrunner-server/app-logger/v6"
	configImpl "github.com/roadrunner-server/config/v6"
	"github.com/roadrunner-server/endure/v2"
	"github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

// newAppLoggerClient builds an h2c Connect client for the migrated
// applogger.v2.AppLoggerService served by the rpc plugin.
func newAppLoggerClient(t *testing.T, address string) apploggerV2connect.AppLoggerServiceClient {
	t.Helper()
	httpc := &http.Client{Transport: &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return new(net.Dialer).DialContext(ctx, network, addr)
		},
	}}
	t.Cleanup(httpc.CloseIdleConnections)
	return apploggerV2connect.NewAppLoggerServiceClient(httpc, "http://"+address)
}

func TestAppLogger(t *testing.T) {
	container := endure.New(slog.LevelDebug)

	vp := &configImpl.Plugin{}
	vp.Path = "configs/.rr-appl.yaml"
	vp.Version = "v2023.1.0"

	l, oLogger := mocklogger.SlogTestLogger(slog.LevelDebug)
	err := container.RegisterAll(
		&rpc.Plugin{},
		&applogger.Plugin{},
		l,
		vp,
	)
	require.NoError(t, err)

	require.NoError(t, container.Init())

	ch, err := container.Serve()
	require.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				if err := container.Stop(); err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-sig:
				if err := container.Stop(); err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				if err := container.Stop(); err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second)

	client := newAppLoggerClient(t, "127.0.0.1:6001")
	ctx := t.Context()

	_, err = client.Debug(ctx, connect.NewRequest(&v2.LogMessage{Message: "Debug message"}))
	require.NoError(t, err)
	_, err = client.Error(ctx, connect.NewRequest(&v2.LogMessage{Message: "Error message"}))
	require.NoError(t, err)
	_, err = client.Info(ctx, connect.NewRequest(&v2.LogMessage{Message: "Info message"}))
	require.NoError(t, err)
	_, err = client.Warning(ctx, connect.NewRequest(&v2.LogMessage{Message: "Warning message"}))
	require.NoError(t, err)

	time.Sleep(time.Second)
	stopCh <- struct{}{}
	wg.Wait()

	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Debug message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Error message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Info message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Warning message").Len())
}

func TestAppLoggerWithContext(t *testing.T) {
	container := endure.New(slog.LevelDebug)

	vp := &configImpl.Plugin{}
	vp.Path = "configs/.rr-appl-context.yaml"
	vp.Version = "v2023.1.0"

	l, oLogger := mocklogger.SlogTestLogger(slog.LevelDebug)
	err := container.RegisterAll(
		&rpc.Plugin{},
		&applogger.Plugin{},
		l,
		vp,
	)
	require.NoError(t, err)

	require.NoError(t, container.Init())

	ch, err := container.Serve()
	require.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				if err := container.Stop(); err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-sig:
				if err := container.Stop(); err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				if err := container.Stop(); err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second)

	client := newAppLoggerClient(t, "127.0.0.1:6002")
	ctx := t.Context()

	entries := []struct {
		method func(context.Context, *connect.Request[v2.LogEntry]) (*connect.Response[v2.LogResponse], error)
		entry  *v2.LogEntry
	}{
		{client.DebugWithContext, &v2.LogEntry{Message: "Debug context message", LogAttrs: []*v2.LogAttrs{{Key: "component", Value: "test"}}}},
		{client.ErrorWithContext, &v2.LogEntry{Message: "Error context message", LogAttrs: []*v2.LogAttrs{{Key: "error_code", Value: "500"}, {Key: "trace", Value: "stack_trace_here"}}}},
		{client.InfoWithContext, &v2.LogEntry{Message: "Info context message", LogAttrs: []*v2.LogAttrs{{Key: "request_id", Value: "12345"}, {Key: "user", Value: "john"}}}},
		{client.WarningWithContext, &v2.LogEntry{Message: "Warning context message", LogAttrs: []*v2.LogAttrs{{Key: "threshold", Value: "90"}}}},
	}
	for _, e := range entries {
		_, err = e.method(ctx, connect.NewRequest(e.entry))
		require.NoError(t, err)
	}

	time.Sleep(time.Second)
	stopCh <- struct{}{}
	wg.Wait()

	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Debug context message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Error context message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Info context message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Warning context message").Len())

	assert.Equal(t, 1, oLogger.FilterAttr("component", "test").Len())
	assert.Equal(t, 1, oLogger.FilterAttr("request_id", "12345").Len())
	assert.Equal(t, 1, oLogger.FilterAttr("error_code", "500").Len())
	assert.Equal(t, 1, oLogger.FilterAttr("trace", "stack_trace_here").Len())
	assert.Equal(t, 1, oLogger.FilterAttr("user", "john").Len())
	assert.Equal(t, 1, oLogger.FilterAttr("threshold", "90").Len())
}
