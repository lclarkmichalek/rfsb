package rfsb

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Resource is a resource that can be materialized by rfsb
type Resource interface {
	Materialize(context.Context) error

	resourceMeta() *ResourceMeta
}

// SkippableResource should be implemented by resources that can check to see if they need to be materialized, and skip materialisation if not needed.
type SkippableResource interface {
	Resource
	ShouldSkip(context.Context) (bool, error)
}

// ResourceMeta is struct that should be embedded by all Resource implementers, so as to simplify the accounting side of things
type ResourceMeta struct {
	Name   string
	Logger *logrus.Entry
	// Resources that should be run when this resource has finished running
	dependsOnThis []Resource
	thisDependsOn []Resource
}

func (rm *ResourceMeta) resourceMeta() *ResourceMeta {
	return rm
}

// Called when the resource is registered with the registry
func (rm *ResourceMeta) initialize(name string) {
	rm.Name = name
	rm.Logger = logrus.New().WithField("resource", name)
}

func Run(r Resource) *DependencySetter {
	return &DependencySetter{target: r}
}

type DependencySetter struct {
	target Resource
	source Resource
}

func (ds *DependencySetter) When(r Resource) *DependencySetter {
	ds.source = r
	return ds
}

func (ds *DependencySetter) Finishes() {
	targetRM := ds.target.resourceMeta()
	sourceRM := ds.source.resourceMeta()

	sourceRM.dependsOnThis = append(sourceRM.dependsOnThis, ds.target)
	targetRM.thisDependsOn = append(targetRM.thisDependsOn, ds.source)
}
