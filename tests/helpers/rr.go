// Package helpers boots the RoadRunner container the integration tests run against.
package helpers

import (
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	mocklogger "tests/mock"

	"github.com/roadrunner-server/config/v6"
	"github.com/roadrunner-server/endure/v2"
	"github.com/stretchr/testify/require"
)

const (
	// configVersion is the schema version the configs under tests/configs declare.
	configVersion = "v2023.1.0"
	// probeTimeout caps how long Start waits for the rpc listener to answer.
	probeTimeout = time.Second * 15
	probeTick    = time.Millisecond * 20
	probeDial    = time.Second
)

// RR is a running container.
type RR struct {
	// Logs holds the records captured by the in-memory logger the container runs with.
	Logs *mocklogger.ObservedLogs
}

// Start registers the plugins, boots the container and returns once rpcAddr
// accepts a connection. The rpc plugin binds its listener in Serve, so a
// successful dial proves the RPC methods are reachable; the address is a
// parameter rather than an option so no test can skip the wait.
//
// Errors arriving on the container channel are reported through t.Errorf and
// stop the container, but they do not abort the test.
//
// The returned stop is idempotent and also registered with t.Cleanup, so a
// failed require between boot and the assertions still releases the port.
func Start(t *testing.T, cfgPath, rpcAddr string, plugins []any) (*RR, func()) {
	t.Helper()

	cfg := &config.Plugin{Path: cfgPath, Version: configVersion}

	l, obs := mocklogger.SlogTestLogger(slog.LevelDebug)
	rr := &RR{Logs: obs}

	cont := endure.New(slog.LevelDebug)
	require.NoError(t, cont.RegisterAll(append([]any{cfg, l}, plugins...)...))
	require.NoError(t, cont.Init())

	ch, err := cont.Serve()
	require.NoError(t, err)

	stopCont := sync.OnceValue(cont.Stop)
	done := make(chan struct{})
	wg := &sync.WaitGroup{}

	wg.Go(func() {
		for {
			select {
			case res := <-ch:
				t.Errorf("plugin %s reported an error: %v", res.VertexID, res.Error)
				if errS := stopCont(); errS != nil {
					t.Errorf("container stop: %v", errS)
				}
			case <-done:
				if errS := stopCont(); errS != nil {
					t.Errorf("container stop: %v", errS)
				}
				return
			}
		}
	})

	// The drain goroutine calls t.Errorf, so it has to be joined while the test
	// is still running.
	stop := sync.OnceFunc(func() {
		close(done)
		wg.Wait()
	})
	t.Cleanup(stop)

	require.Eventually(t, func() bool {
		d := net.Dialer{Timeout: probeDial}
		conn, errD := d.DialContext(t.Context(), "tcp", rpcAddr)
		if errD != nil {
			return false
		}

		_ = conn.Close()
		return true
	}, probeTimeout, probeTick, "container did not become ready")

	return rr, stop
}
