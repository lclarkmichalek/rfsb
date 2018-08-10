package rfsb

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// ResourceGraph is a container for Resources and dependencies between them.
type ResourceGraph struct {
	ResourceMeta

	resources           []Resource
	dependencies        map[Resource]map[Resource][]Signal
	inverseDependencies map[Resource]map[Resource][]Signal
}

func (rg *ResourceGraph) init() {
	if len(rg.dependencies) == 0 {
		rg.dependencies = map[Resource]map[Resource][]Signal{}
	}
	if len(rg.inverseDependencies) == 0 {
		rg.inverseDependencies = map[Resource]map[Resource][]Signal{}
	}
}

// Register adds the Resource to the graph.
//
// The Resource will have its Initialize method called with the passed name.
func (rg *ResourceGraph) Register(name string, r Resource) {
	rg.init()
	r.SetName(name)

	if otherRG, ok := r.(*ResourceGraph); ok {
		rg.resources = append(rg.resources, otherRG.resources...)
		for from, tos := range otherRG.dependencies {
			for to, signals := range tos {
				if _, ok := rg.dependencies[from]; !ok {
					rg.dependencies[from] = map[Resource][]Signal{to: signals}
				} else {
					rg.dependencies[from][to] = append(rg.dependencies[from][to], signals...)
				}
			}
		}
		for to, froms := range otherRG.inverseDependencies {
			for from, signals := range froms {
				if _, ok := rg.inverseDependencies[to]; !ok {
					rg.inverseDependencies[to] = map[Resource][]Signal{from: signals}
				} else {
					rg.inverseDependencies[to][from] = append(rg.inverseDependencies[to][from], signals...)
				}
			}
		}
	} else {
		rg.resources = append(rg.resources, r)
	}
}

// RegisterDependency adds a dependency between two resources. This ensures the second resource (`to`) will not be
// Materialized before the first resource (`from`) has finished Materializing
func (rg *ResourceGraph) RegisterDependency(from Resource, signal Signal, to Resource) {
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
			if _, ok := rg.dependencies[from]; !ok {
				rg.dependencies[from] = map[Resource][]Signal{}
			}
			rg.dependencies[from][to] = append(rg.dependencies[from][to], signal)
			if _, ok := rg.inverseDependencies[to]; !ok {
				rg.inverseDependencies[to] = map[Resource][]Signal{}
			}
			rg.inverseDependencies[to][from] = append(rg.inverseDependencies[to][from], signal)
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

func (rg *ResourceGraph) waitForSignals(ctx context.Context, ch <-chan Signal, sigs []Signal) (met bool, unemittedButExpected []Signal) {
	expectedSignals := map[Signal]struct{}{}
	for _, expectedSig := range sigs {
		expectedSignals[expectedSig] = struct{}{}
	}

	for {
		var sig Signal
		select {
		case <-ctx.Done():
			return false, nil
		case sig = <-ch:
		}
		_, found := expectedSignals[sig]
		if found {
			delete(expectedSignals, sig)
		}
		if len(expectedSignals) == 0 {
			return true, nil
		}
		if sig == Finished {
			for s := range expectedSignals {
				unemittedButExpected = append(unemittedButExpected, s)
			}
			return false, unemittedButExpected
		}
	}
}

// Materialize executes all of the resources in the resource graph. Resources will be materialized in parallel, while
// not violating constraints introduced by RegisterDependency
func (rg *ResourceGraph) Materialize(ctx context.Context) error {
	dependencyChans := make(map[Resource]map[Resource]chan Signal, len(rg.resources))
	for to, froms := range rg.inverseDependencies {
		dependencyChans[to] = make(map[Resource]chan Signal, len(froms))
		for from := range froms {
			dependencyChans[to][from] = make(chan Signal, 3)
		}
	}

	grp, ctx := errgroup.WithContext(ctx)
	for _, resource := range rg.resources {
		resource := resource
		emit := func(sig Signal) {
			for to := range rg.dependencies[resource] {
				dependencyChans[to][resource] <- sig
			}
		}

		grp.Go(func() error {
			defer emit(Finished)

			var dependenciesFinishedButNotMet uint32
			wg := sync.WaitGroup{}
			for from, ch := range dependencyChans[resource] {
				from := from
				ch := ch

				wg.Add(1)
				go func() {
					defer wg.Done()
					met, unemitted := rg.waitForSignals(ctx, ch, rg.inverseDependencies[resource][from])
					if met {
						return
					}
					if len(unemitted) != 0 {
						strUnemitted := []string{}
						for _, s := range unemitted {
							strUnemitted = append(strUnemitted, s.String())
						}
						resource.Logger().Infof(
							"skipping due to %s finishing without emitting %s",
							from.Name(),
							strings.Join(strUnemitted, ", "))
					}
					atomic.AddUint32(&dependenciesFinishedButNotMet, 1)
				}()
			}
			wg.Wait()
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if dependenciesFinishedButNotMet != 0 {
				defer emit(Unevaluated)
				return nil
			}
			defer emit(Evaluated)

			var shouldSkip bool
			started := time.Now()
			resource.Logger().Infof("evaluating resource")
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
				defer emit(Materialized)
			} else {
				resource.Logger().Infof("skipping resource materialization")
				defer emit(Skipped)
			}
			resource.Logger().Infof("resource evaluated in %v", time.Now().Sub(started))
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

func (rg *ResourceGraph) String() string {
	buf := &bytes.Buffer{}
	buf.WriteString("ResourceGraph{rs:[")
	for i, resource := range rg.resources {
		buf.WriteString(resource.Name())
		if i != len(rg.resources)-1 {
			buf.WriteString(" ")
		}
	}
	buf.WriteString("], deps:{")
	i := 0
	for from, tos := range rg.dependencies {
		for to := range tos {
			buf.WriteString(from.Name())
			buf.WriteString(":")
			buf.WriteString(to.Name())
			i++
			if i != len(rg.dependencies) {
				buf.WriteString(" ")
			}
		}
	}
	buf.WriteString("}}")
	return buf.String()
}

// When returns a new DependencySetter for the ResourceGraph.
//
// When is the main API that should be used to register dependencies. It's most basic use is just simple chaining of
// dependencies:
func (rg *ResourceGraph) When(resource Resource, signals ...Signal) *DependencySetter {
	if len(signals) == 0 {
		signals = []Signal{Evaluated}
	}
	deps := []dependency{}
	for _, sig := range signals {
		deps = append(deps, dependency{resource, sig})
	}
	return &DependencySetter{
		registry:     rg,
		dependencies: deps,
	}
}

type registry interface {
	Register(string, Resource)
	RegisterDependency(from Resource, signal Signal, to Resource)
}

// DependencySetter is a helper to improve the ergonomics of creating dependencies between resources
type DependencySetter struct {
	registry     registry
	dependencies []dependency
}

type dependency struct {
	resource Resource
	signal   Signal
}

// And adds an additional resource that must be completed before the Resource passed to Do will be run
func (ds *DependencySetter) And(source Resource, signals ...Signal) *DependencySetter {
	if len(signals) == 0 {
		signals = []Signal{Evaluated}
	}
	for _, signal := range signals {
		ds.dependencies = append(ds.dependencies, dependency{source, signal})
	}
	return ds
}

// Do registers the passed Resource as being dependent on the Resources passed to When and And
func (ds *DependencySetter) Do(name string, target Resource) *DependencySetter {
	ds.registry.Register(name, target)
	for _, dependency := range ds.dependencies {
		ds.registry.RegisterDependency(dependency.resource, dependency.signal, target)
	}
	return ds
}
