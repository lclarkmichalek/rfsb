package rfsb

import "context"

var (
	// DefaultRegistry is a default instance of the ResourceRegistry available for convenience
	DefaultRegistry = ResourceRegistry{}
)

// Register is the Register method on the DefaultRegistry
func Register(name string, r Resource) {
	DefaultRegistry.Register(name, r)
}

func Validate() error {
	return DefaultRegistry.Validate()
}

func MaterializeChanges(ctx context.Context) error {
	return DefaultRegistry.MaterializeChanges(ctx)
}
