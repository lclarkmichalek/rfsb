package rfsb

import "context"

var (
	// DefaultRegistry is an instance of the ResourceGraph that a number of convenience functions act on.
	DefaultRegistry = ResourceGraph{}
)

// Register registers the resource on the DefaultRegistry with the passed name
//
// Register is a shortcut for DefaultRegistry.Register. See there for more details
func Register(name string, r Resource) {
	DefaultRegistry.Register(name, r)
}

// When creates a DependencySetter using the DefaultRegistry
//
// When is a shortcut for DefaultRegistry.When. See there for more details
func When(source Resource) *DependencySetter {
	return DefaultRegistry.When(source)
}

// Materialize materializes the resources registered with the DefaultRegistry
//
// Materialize is a shortcut for DefaultRegistry.Materialize. See there for more details
func Materialize(ctx context.Context) error {
	return DefaultRegistry.Materialize(ctx)
}
