package rfsb

import "os"

func ExampleResourceGraph() {
	rg := &ResourceGraph{}

	firstFile := &FileResource{
		Path:     "/tmp/first_file",
		Contents: "hello",
		Mode:     0644,
		UID:      uint32(os.Getuid()),
		GID:      uint32(os.Getgid()),
	}
	rg.Register("firstFile", firstFile)

	secondFile := &FileResource{
		Path:     "/tmp/second_file",
		Contents: "world",
		Mode:     0644,
		UID:      uint32(os.Getuid()),
		GID:      uint32(os.Getgid()),
	}
	rg.When(firstFile, SignalFinished).Do("firstFile", secondFile)
}

func ExampleResourceGraph_When() {
	rg := &ResourceGraph{}

	user := &UserResource{
		User:  "test",
		UID:   1999,
		GID:   1999,
		Home:  "/tmp",
		Shell: "/bin/bash",
	}
	rg.Register("user", user)

	group := &GroupResource{
		Group: "bozos",
		GID:   1888,
	}
	rg.Register("group", group)

	membership := &GroupMembershipResource{
		GID:  group.GID,
		User: user.User,
	}
	rg.When(user, SignalFinished).And(group, SignalFinished).Do("membership", membership)
}
