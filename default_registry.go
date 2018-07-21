package rfsb

import "context"

var (
	// DefaultRegistry is a default instance of the ResourceRegistry available for convenience
	DefaultRegistry = ResourceGraph{}
)

// Register is the Register method on the DefaultRegistry
func Register(name string, r Resource) {
	DefaultRegistry.Register(name, r)
}

func When(source Resource) *DependencySetter {
	return DefaultRegistry.When(source)
}

// func Validate() error {
// 	return DefaultRegistry.Validate()
// }

func MaterializeChanges(ctx context.Context) error {
	return DefaultRegistry.Materialize(ctx)
}
