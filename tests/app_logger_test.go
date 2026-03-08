package app_logger //nolint:stylecheck

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	mocklogger "tests/mock"

	applogger "github.com/roadrunner-server/app-logger/v5"
	configImpl "github.com/roadrunner-server/config/v5"
	"github.com/roadrunner-server/endure/v2"
	"github.com/roadrunner-server/http/v5"
	"github.com/roadrunner-server/rpc/v5"
	"github.com/roadrunner-server/server/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAppLogger(t *testing.T) {
	container := endure.New(slog.LevelDebug)

	vp := &configImpl.Plugin{}
	vp.Path = "configs/.rr-appl.yaml"
	vp.Version = "v2023.1.0"

	l, oLogger := mocklogger.ZapTestLogger(zap.DebugLevel)
	err := container.RegisterAll(
		&rpc.Plugin{},
		&applogger.Plugin{},
		l,
		&server.Plugin{},
		&http.Plugin{},
		vp,
	)

	require.NoError(t, err)

	err = container.Init()
	require.NoError(t, err)

	ch, err := container.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = container.Stop()

				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-sig:
				err = container.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = container.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second * 2)
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

	l, oLogger := mocklogger.ZapTestLogger(zap.DebugLevel)
	err := container.RegisterAll(
		&rpc.Plugin{},
		&applogger.Plugin{},
		l,
		&server.Plugin{},
		&http.Plugin{},
		vp,
	)

	require.NoError(t, err)

	err = container.Init()
	require.NoError(t, err)

	ch, err := container.Serve()
	assert.NoError(t, err)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopCh := make(chan struct{}, 1)

	wg.Go(func() {
		for {
			select {
			case e := <-ch:
				assert.Fail(t, "error", e.Error.Error())
				err = container.Stop()

				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-sig:
				err = container.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			case <-stopCh:
				// timeout
				err = container.Stop()
				if err != nil {
					assert.FailNow(t, "error", err.Error())
				}
				return
			}
		}
	})

	time.Sleep(time.Second * 2)
	stopCh <- struct{}{}

	wg.Wait()

	// Verify context messages were captured
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Debug context message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Error context message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Info context message").Len())
	assert.Equal(t, 1, oLogger.FilterMessageSnippet("Warning context message").Len())

	// Verify context fields are present
	assert.Equal(t, 1, oLogger.FilterField(zap.String("component", "test")).Len())
	assert.Equal(t, 1, oLogger.FilterField(zap.String("request_id", "12345")).Len())
	assert.Equal(t, 1, oLogger.FilterField(zap.String("error_code", "500")).Len())
	assert.Equal(t, 1, oLogger.FilterField(zap.String("trace", "stack_trace_here")).Len())
	assert.Equal(t, 1, oLogger.FilterField(zap.String("user", "john")).Len())
	assert.Equal(t, 1, oLogger.FilterField(zap.String("threshold", "90")).Len())
}
