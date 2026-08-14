package app_logger //nolint:stylecheck

// The two containers the integration tests boot, one config per test file. Each
// address is the rpc.listen value of the config next to it, and the port is
// unique to this repo so a concurrent run of a sibling plugin cannot take it.
const (
	levelsConfig  = "configs/.rr-appl.yaml"
	levelsRPCAddr = "127.0.0.1:6301"

	contextConfig  = "configs/.rr-appl-context.yaml"
	contextRPCAddr = "127.0.0.1:6302"
)
