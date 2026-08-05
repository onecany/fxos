package trader

import "fxos/trader/types"

// SafeFloat64 / SafeString / SafeInt are thin wrappers around the
// implementations in trader/types (safe.go) so exchange response shape
// changes degrade gracefully instead of panicking. Both the engine package
// (via these wrappers) and the exchange subpackages (via types.Safe*) share
// the same implementation without an import cycle.
func SafeFloat64(data map[string]interface{}, key string) (float64, error) {
	return types.SafeFloat64(data, key)
}

func SafeString(data map[string]interface{}, key string) (string, error) {
	return types.SafeString(data, key)
}

func SafeInt(data map[string]interface{}, key string) (int, error) {
	return types.SafeInt(data, key)
}
