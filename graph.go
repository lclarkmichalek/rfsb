package rfsb

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type ResourceGraph struct {
	name                string
	logger              *logrus.Entry
	resources           []Resource
	dependencies        map[Resource][]Resource
	inverseDependencies map[Resource][]Resource
}

func (rg *ResourceGraph) init() {
	rg.dependencies = map[Resource][]Resource{}
	rg.inverseDependencies = map[Resource][]Resource{}
}

func (rg *ResourceGraph) Register(name string, r Resource) {
	rg.init()
	r.Initialize(name)

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

func (rg *ResourceGraph) merge(with *ResourceGraph) {
	rg.resources = append(rg.resources, with.resources...)
	for _, resource := range with.resources {
		rg.dependencies[resource] = append(rg.dependencies[resource], with.dependencies[resource]...)
		rg.inverseDependencies[resource] = append(rg.inverseDependencies[resource], with.inverseDependencies[resource]...)
	}
}

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
					return errors.Wrap(err, "could not determine if materialization should be skipped")
				}
			}
			if !shouldSkip {
				resource.Logger().Infof("materializing resource")
				err := resource.Materialize(ctx)
				if err != nil {
					return errors.Wrap(err, "could not materialize resource")
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

func (rg *ResourceGraph) When(sources ...Resource) *DependencySetter {
	return &DependencySetter{
		sources:  sources,
		registry: rg,
	}
}

func (rg *ResourceGraph) Initialize(name string) {
	rg.name = name
	rg.logger = logrus.New().WithField("graph", name)
}
func (rg *ResourceGraph) Name() string {
	return rg.name
}
func (rg *ResourceGraph) Logger() *logrus.Entry {
	return rg.logger
}
