package rfsb

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Resource is a resource that can be materialized by rfsb
type Resource interface {
	Materialize(context.Context, chan<- Signal) error

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

// Name returns the name of the Resource
func (rm *ResourceMeta) Name() string {
	return rm.name
}

// Logger returns the Resource's logger
func (rm *ResourceMeta) Logger() *logrus.Entry {
	return rm.logger
}

// SetName is called when the resource is registered with the registry
func (rm *ResourceMeta) SetName(name string) {
	rm.name = name
	rm.logger = logrus.New().WithField("resource", name)
}
