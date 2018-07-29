package rfsb

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Resource is a resource that can be materialized by rfsb
type Resource interface {
	Materialize(context.Context) error

	// Best to implement these by embedding ResourceMeta
	Name() string
	SetName(string)
	Logger() *logrus.Entry
}

// ResourceMeta is struct that should be embedded by all Resource implementers, so as to simplify the accounting side of things
type ResourceMeta struct {
	name   string
	logger *logrus.Entry
}

func (rm *ResourceMeta) Name() string {
	return rm.name
}

func (rm *ResourceMeta) Logger() *logrus.Entry {
	return rm.logger
}

// SetName is called when the resource is registered with the registry
func (rm *ResourceMeta) SetName(name string) {
	rm.name = name
	rm.logger = logrus.New().WithField("resource", name)
}

type registry interface {
	Register(string, Resource)
	RegisterDependency(from, to Resource)
}

type DependencySetter struct {
	registry registry
	sources  []Resource
}

func (ds *DependencySetter) And(source Resource) *DependencySetter {
	ds.sources = append(ds.sources, source)
	return ds
}

func (ds *DependencySetter) Do(name string, target Resource) *DependencySetter {
	ds.registry.Register(name, target)
	for _, source := range ds.sources {
		ds.registry.RegisterDependency(source, target)
	}
	return ds
}
