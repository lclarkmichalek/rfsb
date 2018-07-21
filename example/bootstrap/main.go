package main

import (
	"context"

	"github.com/lclarkmichalek/rfsb"
	"github.com/sirupsen/logrus"
)

func addUsers() *rfsb.ResourceGraph {
	rg := &rfsb.ResourceGraph{}

	addLCMGroup := &rfsb.GroupResource{
		Group: "lcm",
		GID:   1000,
	}
	rg.Register("lcmGroup", addLCMGroup)

	addLCM := &rfsb.UserResource{
		User:  "lcm",
		UID:   1000,
		GID:   1000,
		Home:  "/home/lcm",
		Shell: "/bin/bash",
	}
	rg.When(addLCMGroup).Do("lcm", addLCM)

	for group, gid := range map[string]uint32{"sudo": 150, "docker": 233} {
		addLCMToGroup := &rfsb.GroupMembershipResource{
			User: addLCM.User,
			GID:  gid,
		}
		rg.When(addLCM).Do("lcmGroupMembership#"+group, addLCMToGroup)
	}

	addLCMSSHKey := &rfsb.FileResource{
		Path:     "/home/lcm/.ssh/authorized_keys",
		Mode:     0400,
		Contents: `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAL5YH0a+pKd8E8Be97+gN/kn+U71JCapIH8uysrecKB lcm@lcm-mbp`,
		UID:      addLCM.UID,
		GID:      addLCMGroup.GID,
	}
	rg.When(addLCM).Do("addLCM", addLCMSSHKey)

	addInputRC := &rfsb.FileResource{
		Path:     "/home/lcm/.inputrc",
		Mode:     0644,
		Contents: `set editing-mode vi`,
		UID:      addLCM.UID,
		GID:      addLCMGroup.GID,
	}
	rg.When(addLCM).Do("addInputRC", addInputRC)

	return rg
}

func mongodb() *rfsb.ResourceGraph {
	rg := &rfsb.ResourceGraph{}

	group := &rfsb.GroupResource{
		Group: "mongodb",
		GID:   1001,
	}
	rg.Register("mongodbGroup", group)

	user := &rfsb.UserResource{
		User:  "mongodb",
		UID:   1001,
		GID:   1001,
		Home:  "/var/lib/mongodb-current",
		Shell: "/bin/bash",
	}
	rg.Register("mongodbUser", user)

	serviceFile := &rfsb.FileResource{
		Path: "/etc/systemd/system/mongodb.service",
		Mode: 0644,
		Contents: `
[Unit]
Description=the mongodb database

[Service]
ExecStartPre=+/usr/bin/rm -rf /var/lib/mongodb-current
ExecStartPre=+/usr/bin/cp -r /var/lib/mongodb-next /var/lib/mongodb-current
ExecStartPre=+/usr/bin/install -o mongodb -g mongodb -d -m 0755 /var/run/mongodb
ExecStartPre=+/usr/bin/install -o mongodb -g mongodb -d -m 0700 /var/lib/mongodb-data
ExecStart=/var/lib/mongodb-current/mongod \
  --dbpath=/var/lib/mongodb-data \
  --auth \
  --unixSocketPrefix=/var/run/mongodb

User=mongodb
Group=mongodb
NoNewPrivileges=True
ProtectSystem=True
ProtectHome=True
ProtectKernelTunables=True
ProtectKernelModules=True

StandardOutput=journal
StandardError=journal`,
		UID: user.UID,
		GID: user.GID,
	}
	rg.When(user).And(group).Do("serviceFile", serviceFile)

	return rg
}

func main() {
	rfsb.SetupDefaultLogging()

	addUsers := addUsers()
	rfsb.Register("users", addUsers)

	addMongodb := mongodb()
	rfsb.When(addUsers).Do("mongo", addMongodb)

	// err := rfsb.Validate()
	// if err != nil {
	// 	logrus.Fatal("failed to validate resource graph: ", err)
	// }

	err := rfsb.MaterializeChanges(context.Background())
	if err != nil {
		logrus.Fatal("failed to materialize changes: ", err)
	}
}
