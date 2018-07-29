package rfsb

import (
	"context"
)

// SkippableResource should be implemented by resources that can check to see if they need to be materialized, and skip materialisation if not needed.
type SkippableResource interface {
	Resource
	ShouldSkip(context.Context) (bool, error)
}

// SkippingWrapper wraps a resource, allowing the user to provide a custom ShouldSkip function
type SkippingWrapper struct {
	Resource
	SkipFunc func(context.Context) (bool, error)
}

// ShouldSkip calls the provided SkipFunc. If the underlying resource is a SkippableResource, its ShouldSkip will also
// be called, making the SkipFunc additive.
func (sw *SkippingWrapper) ShouldSkip(ctx context.Context) (bool, error) {
	shouldSkip, err := sw.SkipFunc(ctx)
	if err != nil || shouldSkip {
		return shouldSkip, err
	}

	if sr, ok := sw.Resource.(SkippableResource); ok {
		return sr.ShouldSkip(ctx)
	}

	return false, nil
}
