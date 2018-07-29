package rfsb

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResourceGraphFlattening tests that when we flatten a ResourceGraph into another ResourceGraph, all of the
// resources are preserved
func TestResourceGraphFlattening(t *testing.T) {
	t.Parallel()

	innerRG := &ResourceGraph{}
	r1 := mkArbitraryResource(t)
	r2 := mkArbitraryResource(t)
	innerRG.Register("r1", r1)
	innerRG.Register("r2", r2)

	outerRG := &ResourceGraph{}
	outerRG.Register("inner", innerRG)

	assert.ElementsMatch(t, outerRG.resources, innerRG.resources)
}

// TestResourceGraphFlattening tests that when we flatten a ResourceGraph into another ResourceGraph, all of the
// dependencies between resources are preserved
func TestResourceGraphFlatteningDependencies(t *testing.T) {
	t.Parallel()

	innerRG := &ResourceGraph{}
	r1 := mkArbitraryResource(t)
	r2 := mkArbitraryResource(t)
	innerRG.Register("r1", r1)
	innerRG.When(r1).Do("r2", r2)

	outerRG := &ResourceGraph{}
	outerRG.Register("inner", innerRG)

	assert.ElementsMatch(t, outerRG.resources, innerRG.resources)
	deps, ok := outerRG.dependencies[r1]
	assert.Equal(t, ok, true)
	assert.ElementsMatch(t, deps, []Resource{r2})
	inverseDeps, ok := outerRG.inverseDependencies[r2]
	assert.Equal(t, ok, true)
	assert.ElementsMatch(t, inverseDeps, []Resource{r1})
}

func mkArbitraryResource(t *testing.T) Resource {
	scratchDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Skipf("could not create test dir: %v", err)
	}

	number := rand.Uint64()
	return &FileResource{
		Path:     fmt.Sprintf("%v/%v.foo", scratchDir, number),
		Contents: "foo",
		Mode:     0644,
		UID:      uint32(os.Getuid()),
		GID:      uint32(os.Getgid()),
	}
}
