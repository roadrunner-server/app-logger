package app_logger //nolint:stylecheck

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"tests/helpers"

	applogger "github.com/roadrunner-server/app-logger/v6"
	rpcPlugin "github.com/roadrunner-server/rpc/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// phpClientDir holds the composer project and the client scripts.
const phpClientDir = "php_test_files"

// strictEnv turns a missing piece of the PHP toolchain from a skip into a
// failure. CI installs php with ext-sockets and runs composer, and sets this, so
// the only cross-language tests in the repo cannot vanish from a green run.
const strictEnv = "RR_APP_LOGGER_REQUIRE_PHP"

// phpClient returns the php binary to run the client scripts with, or skips the
// test when the machine cannot run them: goridge talks to the rpc plugin over
// ext-sockets, and the scripts need the composer dependencies on disk.
func phpClient(t *testing.T) string {
	t.Helper()

	bin, err := exec.LookPath("php")
	if err != nil {
		skipOrFailf(t, "php not found in PATH")
		return ""
	}

	if _, err = os.Stat(filepath.Join(phpClientDir, "vendor", "autoload.php")); err != nil {
		skipOrFailf(t, "composer dependencies are not installed in %s", phpClientDir)
		return ""
	}

	probe := exec.CommandContext(t.Context(), bin, "-r", `exit(extension_loaded($argv[1]) ? 0 : 1);`, "--", "sockets")
	if err = probe.Run(); err != nil {
		skipOrFailf(t, "php was built without the sockets extension")
		return ""
	}

	return bin
}

// skipOrFailf skips the test, or fails it when the strict env var is set.
func skipOrFailf(t *testing.T, format string, args ...any) {
	t.Helper()

	if os.Getenv(strictEnv) == "1" {
		require.Failf(t, "the php test environment is incomplete", format, args...)
		return
	}

	t.Skipf(format+" (set %s=1 to fail instead of skipping)", append(args, strictEnv)...)
}

// runPHPClient runs a client script to completion. Every RPC call it makes is
// synchronous, so once the process is gone the records it produced are already
// in the observed logs.
func runPHPClient(t *testing.T, bin, script string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), bin, script)
	cmd.Dir = phpClientDir

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s output: %s", script, out)
}

func TestPHPClientLevels(t *testing.T) {
	bin := phpClient(t)

	rr, _ := helpers.Start(t, levelsConfig, levelsRPCAddr,
		[]any{&rpcPlugin.Plugin{}, &applogger.Plugin{}})

	runPHPClient(t, bin, "appLogger.php")

	records := []struct {
		message string
		level   slog.Level
	}{
		{"Debug message", slog.LevelDebug},
		{"Error message", slog.LevelError},
		{"Info message", slog.LevelInfo},
		{"Warning message", slog.LevelWarn},
	}

	for _, r := range records {
		assert.Equal(t, 1, rr.Logs.FilterLevelExact(r.level).FilterMessageSnippet(r.message).Len(),
			"the php client should produce exactly one %s record", r.level)
	}

	// The client also calls log(), which goes to the process stderr.
	assert.Zero(t, rr.Logs.FilterMessageSnippet("Log message").Len(),
		"the raw log() call should not reach the logger sink")
}

func TestPHPClientWithContext(t *testing.T) {
	bin := phpClient(t)

	rr, _ := helpers.Start(t, contextConfig, contextRPCAddr,
		[]any{&rpcPlugin.Plugin{}, &applogger.Plugin{}})

	runPHPClient(t, bin, "appLoggerWithContext.php")

	records := []struct {
		message string
		level   slog.Level
		attrs   map[string]string
	}{
		{"Debug context message", slog.LevelDebug, map[string]string{"component": "test", "request_id": "12345"}},
		{"Error context message", slog.LevelError, map[string]string{"error_code": "500", "trace": "stack_trace_here"}},
		{"Info context message", slog.LevelInfo, map[string]string{"user": "john"}},
		{"Warning context message", slog.LevelWarn, map[string]string{"threshold": "90"}},
	}

	for _, r := range records {
		assert.Equal(t, 1, rr.Logs.FilterLevelExact(r.level).FilterMessageSnippet(r.message).Len(),
			"the php client should produce exactly one %s record", r.level)

		for key, value := range r.attrs {
			assert.Equal(t, 1, rr.Logs.FilterAttr(key, value).Len(),
				"the %s record should carry attr %s", r.level, key)
		}
	}

	assert.Zero(t, rr.Logs.FilterMessageSnippet("Log context message").Len(),
		"the raw log() call should not reach the logger sink")
}
