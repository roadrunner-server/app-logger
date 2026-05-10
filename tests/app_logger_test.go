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
	httpc := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return new(net.Dialer).DialContext(ctx, network, addr)
			},
		},
	}
	t.Cleanup(httpc.CloseIdleConnections)
	return apploggerV2connect.NewAppLoggerServiceClient(httpc, "http://"+address)
}

// waitForRPC polls the rpc plugin's listener until it accepts a TCP connection,
// up to the given deadline. Replaces fragile time.Sleep-based readiness waits.
func waitForRPC(t *testing.T, address string) {
	t.Helper()
	d := &net.Dialer{Timeout: 200 * time.Millisecond}
	require.Eventually(t, func() bool {
		conn, err := d.DialContext(t.Context(), "tcp", address)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 100*time.Millisecond, "rpc plugin did not become ready at %s", address)
}

// serveContainer starts the Endure container and returns a stop function.
// The stop function signals the watcher goroutine, waits for it to exit, and
// is safe to call exactly once at the end of a test (or via t.Cleanup).
// While running, the watcher fails the test on container errors or OS signals.
func serveContainer(t *testing.T, container *endure.Endure) func() {
	t.Helper()
	ch, err := container.Serve()
	require.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		stop := func() {
			if err := container.Stop(); err != nil {
				assert.FailNow(t, "error", err.Error())
			}
		}
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				stop()
				return
			case <-sig:
				stop()
				return
			case <-stopCh:
				stop()
				return
			}
		}
	})

	return func() {
		stopCh <- struct{}{}
		wg.Wait()
	}
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
	stop := serveContainer(t, container)

	waitForRPC(t, "127.0.0.1:6001")

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
	stop()

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
	stop := serveContainer(t, container)

	waitForRPC(t, "127.0.0.1:6002")

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
	stop()

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
