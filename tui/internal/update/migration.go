package update

import (
	"fmt"
	"reflect"
)

type MigrationFunc func(config map[string]any) (map[string]any, error)

type MigrationRegistry struct{ migrations map[int]MigrationFunc }

func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{migrations: map[int]MigrationFunc{}}
}
func (r *MigrationRegistry) Register(fromSchemaVersion int, fn MigrationFunc) {
	if r.migrations == nil {
		r.migrations = map[int]MigrationFunc{}
	}
	if fromSchemaVersion > 0 && fn != nil {
		r.migrations[fromSchemaVersion] = fn
	}
}

func (r *MigrationRegistry) Migrate(config map[string]any, fromVer, toVer int) (map[string]any, error) {
	if fromVer <= 0 || toVer <= 0 {
		return nil, fmt.Errorf("schema versions must be positive")
	}
	if toVer < fromVer {
		return nil, fmt.Errorf("config schema downgrade %d→%d is not supported", fromVer, toVer)
	}
	current := cloneMap(config)
	if fromVer == toVer {
		return current, nil
	}
	if r == nil {
		return nil, fmt.Errorf("missing migration registry for %d→%d", fromVer, toVer)
	}
	for v := fromVer; v < toVer; v++ {
		fn, ok := r.migrations[v]
		if !ok {
			return nil, fmt.Errorf("missing migration %d→%d", v, v+1)
		}
		next, err := fn(cloneMap(current))
		if err != nil {
			return nil, fmt.Errorf("migration %d→%d: %w", v, v+1, err)
		}
		if next == nil {
			return nil, fmt.Errorf("migration %d→%d returned nil config", v, v+1)
		}
		current = next
	}
	return current, nil
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneValue(v)
	}
	return out
}
func cloneValue(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	switch x := v.(type) {
	case map[string]any:
		return cloneMap(x)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = cloneValue(x[i])
		}
		return out
	default:
		return v
	}
}
