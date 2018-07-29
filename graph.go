package rfsb

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// ResourceGraph is a container for Resources and dependencies between them.
type ResourceGraph struct {
	ResourceMeta

	resources           []Resource
	dependencies        map[Resource][]Resource
	inverseDependencies map[Resource][]Resource
}

func (rg *ResourceGraph) init() {
	rg.dependencies = map[Resource][]Resource{}
	rg.inverseDependencies = map[Resource][]Resource{}
}

// Register adds the Resource to the graph.
//
// The Resource will have its Initialize method called with the passed name.
func (rg *ResourceGraph) Register(name string, r Resource) {
	rg.init()
	r.SetName(name)

	if otherRG, ok := r.(*ResourceGraph); ok {
		rg.resources = append(rg.resources, otherRG.resources...)
		for _, resource := range otherRG.resources {
			rg.dependencies[resource] = append(rg.dependencies[resource], otherRG.dependencies[resource]...)
			rg.inverseDependencies[resource] = append(rg.inverseDependencies[resource], otherRG.inverseDependencies[resource]...)
		}
	} else {
		rg.resources = append(rg.resources, r)
	}
}

// RegisterDependency adds a dependency between two resources. This ensures the second resource (`to`) will not be
// Materialized before the first resource (`from`) has finished Materializing
func (rg *ResourceGraph) RegisterDependency(from, to Resource) {
	froms := []Resource{from}
	if fromRG, ok := from.(*ResourceGraph); ok {
		froms = fromRG.leafResources()
	}
	tos := []Resource{to}
	if toRG, ok := to.(*ResourceGraph); ok {
		tos = toRG.rootResources()
	}

	for _, from := range froms {
		for _, to := range tos {
			rg.dependencies[from] = append(rg.dependencies[from], to)
			rg.inverseDependencies[to] = append(rg.inverseDependencies[to], from)
		}
	}
}

// SetName sets the name for the resource, but also prepends the resource name to any registered resources, ensuring
// that the resource hierachy is reflected in the logger naming
func (rg *ResourceGraph) SetName(name string) {
	rg.ResourceMeta.SetName(name)
	for _, r := range rg.resources {
		r.SetName(rg.Name() + "·" + r.Name())
	}
}

// Materialize executes all of the resources in the resource graph. Resources will be materialized in parallel, while
// not violating constraints introduced by RegisterDependency
func (rg *ResourceGraph) Materialize(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dependencyChans := make(map[Resource]chan struct{}, len(rg.resources))
	for _, resource := range rg.resources {
		ch := make(chan struct{}, len(rg.inverseDependencies[resource]))
		dependencyChans[resource] = ch
	}

	grp, ctx := errgroup.WithContext(ctx)
	for _, resource := range rg.resources {
		resource := resource
		dependencyChan := dependencyChans[resource]
		grp.Go(func() (err error) {
			defer func() {
				if err != nil {
					cancel()
				}
			}()
			for range rg.inverseDependencies[resource] {
				select {
				case <-ctx.Done():
					return nil
				case <-dependencyChan:
				}
			}

			started := time.Now()
			resource.Logger().Infof("evaluating resource")
			shouldSkip := false
			if skippable, ok := resource.(SkippableResource); ok {
				var err error
				shouldSkip, err = skippable.ShouldSkip(ctx)
				if err != nil {
					return errors.Wrapf(err, "could not determine if materialization should be skipped for %v", resource.Name())
				}
			}
			if !shouldSkip {
				resource.Logger().Infof("materializing resource")
				err := resource.Materialize(ctx)
				if err != nil {
					return errors.Wrapf(err, "could not materialize resource %v", resource.Name())
				}
			} else {
				resource.Logger().Infof("skipping resource materialization")
			}
			resource.Logger().Infof("resource evaluated in %v", time.Now().Sub(started))

			for _, resource := range rg.dependencies[resource] {
				dependencyChans[resource] <- struct{}{}
			}
			return nil
		})
	}

	return grp.Wait()
}

func (rg *ResourceGraph) rootResources() []Resource {
	roots := []Resource{}
	for _, resource := range rg.resources {
		if _, ok := rg.inverseDependencies[resource]; !ok {
			roots = append(roots, resource)
		}
	}
	return roots
}

func (rg *ResourceGraph) leafResources() []Resource {
	leaves := []Resource{}
	for _, resource := range rg.resources {
		if _, ok := rg.dependencies[resource]; !ok {
			leaves = append(leaves, resource)
		}
	}
	return leaves
}

// When returns a new DependencySetter for the ResourceGraph.
//
// When is the main API that should be used to register dependencies. It's most basic use is just simple chaining of
// dependencies:
func (rg *ResourceGraph) When(sources ...Resource) *DependencySetter {
	return &DependencySetter{
		sources:  sources,
		registry: rg,
	}
}
