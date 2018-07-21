package main

import (
	"context"

	"github.com/lclarkmichalek/rfsb"
	"github.com/sirupsen/logrus"
)

func main() {
	rfsb.SetupDefaultLogging()

	addLCMGroup := &rfsb.GroupResource{
		Group: "lcm",
		GID:   1000,
	}
	rfsb.Register("addLCMGroup", addLCMGroup)

	addLCM := &rfsb.UserResource{
		User:  "lcm",
		UID:   1000,
		GID:   1000,
		Home:  "/home/lcm",
		Shell: "/bin/bash",
	}
	rfsb.Register("addLCM", addLCM)
	rfsb.Run(addLCM).When(addLCMGroup).Finishes()

	for group, gid := range map[string]uint32{"sudo": 150, "docker": 233} {
		addLCMToGroup := &rfsb.GroupMembershipResource{
			User: addLCM.User,
			GID:  gid,
		}
		rfsb.Register("addLCMToGroup#"+group, addLCMToGroup)
		rfsb.Run(addLCMToGroup).When(addLCM).Finishes()
	}

	addLCMSSHKey := &rfsb.FileResource{
		Path:     "/home/lcm/.ssh/authorized_keys",
		Mode:     0400,
		Contents: `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAL5YH0a+pKd8E8Be97+gN/kn+U71JCapIH8uysrecKB lcm@lcm-mbp`,
		UID:      addLCM.UID,
		GID:      addLCMGroup.GID,
	}
	rfsb.Register("addLCMSSHKey", addLCMSSHKey)
	rfsb.Run(addLCMSSHKey).When(addLCM).Finishes()

	err := rfsb.Validate()
	if err != nil {
		logrus.Fatal("failed to validate resource graph: ", err)
	}

	err = rfsb.MaterializeChanges(context.Background())
	if err != nil {
		logrus.Fatal("failed to materialize changes: ", err)
	}
}
