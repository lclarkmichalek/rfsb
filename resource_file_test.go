package rfsb

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
)

var _ SkippableResource = &FileResource{}

func TestFileResourceShouldSkip(t *testing.T) {
	t.Parallel()

	scratchDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Skipf("could not create test dir: %v", err)
	}

	cases := []struct {
		name                 string
		runFirst             *FileResource
		shouldBeSkippable    *FileResource
		shouldNotBeSkippable *FileResource
	}{
		{
			name: "no changes at all",
			runFirst: &FileResource{
				Path:     scratchDir + "/no_change",
				Contents: "hi",
				Mode:     0644,
				UID:      uint32(os.Getuid()),
				GID:      uint32(os.Getgid()),
			},
			shouldBeSkippable: &FileResource{
				Path:     scratchDir + "/no_change",
				Contents: "hi",
				Mode:     0644,
				UID:      uint32(os.Getuid()),
				GID:      uint32(os.Getgid()),
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			c.runFirst.Initialize("runFirst")
			err := c.runFirst.Materialize(context.Background())
			if err != nil {
				t.Errorf("failed to materialize set up resource: %v", err)
				return
			}

			if c.shouldBeSkippable != nil {
				c.shouldBeSkippable.Initialize("shouldBeSkippable")
				shouldSkip, err := c.shouldBeSkippable.ShouldSkip(context.Background())
				if err != nil {
					t.Fatalf("failed to test shouldskip: %v", err)
				}
				if !shouldSkip {
					t.Fatalf("resource was not skippable, should have been")
				}
			}

			if c.shouldNotBeSkippable != nil {
				c.shouldNotBeSkippable.Initialize("shouldNotBeSkippable")
				shouldSkip, err := c.shouldNotBeSkippable.ShouldSkip(context.Background())
				if err != nil {
					t.Fatalf("failed to test shouldskip: %v", err)
				}
				if !shouldSkip {
					t.Fatalf("resource was skippable, should not have been")
				}
			}
		})
	}
}
