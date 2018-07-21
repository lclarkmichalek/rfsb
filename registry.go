package rfsb

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// ResourceRegistry contains all of the resources that will be run
type ResourceRegistry struct {
	resources     []Resource
	resourceMetas []*ResourceMeta
}

type resourceRunNode struct {
	resource              Resource
	waitingOnDependencies chan struct{}
}

func (rr *ResourceRegistry) Register(name string, r Resource) {
	r.resourceMeta().initialize(name)

	rr.resources = append(rr.resources, r)
	rr.resourceMetas = append(rr.resourceMetas, r.resourceMeta())
}

func (rr *ResourceRegistry) Validate() error {
	uniqueNames := map[string]struct{}{}
	for _, rm := range rr.resourceMetas {
		_, ok := uniqueNames[rm.Name]
		if ok {
			return errors.Errorf("duplicate resource named %v", rm.Name)
		}
		uniqueNames[rm.Name] = struct{}{}
	}

	return nil
}

func (rr *ResourceRegistry) MaterializeChanges(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	dependencyChans := make(map[Resource]chan struct{}, len(rr.resources))
	for i, resource := range rr.resources {
		ch := make(chan struct{}, len(rr.resourceMetas[i].thisDependsOn))
		dependencyChans[resource] = ch
	}

	grp, ctx := errgroup.WithContext(ctx)
	for i, resource := range rr.resources {
		resource := resource
		resourceMeta := rr.resourceMetas[i]
		dependencyChan := dependencyChans[resource]
		grp.Go(func() (err error) {
			defer func() {
				if err != nil {
					cancel()
				}
			}()
			for range resourceMeta.thisDependsOn {
				select {
				case <-ctx.Done():
					return nil
				case <-dependencyChan:
				}
			}

			started := time.Now()
			resourceMeta.Logger.Infof("evaluating resource")
			shouldSkip := false
			if skippable, ok := resource.(SkippableResource); ok {
				var err error
				shouldSkip, err = skippable.ShouldSkip(ctx)
				if err != nil {
					return errors.Wrap(err, "could not determine if materialization should be skipped")
				}
			}
			if !shouldSkip {
				resourceMeta.Logger.Infof("materializing resource")
				err := resource.Materialize(ctx)
				if err != nil {
					return errors.Wrap(err, "could not materialize resource")
				}
			} else {
				resourceMeta.Logger.Infof("skipping resource materialization")
			}
			resourceMeta.Logger.Infof("resource evaluated in %v", time.Now().Sub(started))

			for _, resource := range resourceMeta.dependsOnThis {
				dependencyChans[resource] <- struct{}{}
			}
			return nil
		})
	}

	return grp.Wait()
}
