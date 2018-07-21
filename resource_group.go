package rfsb

import (
	"bytes"
	"context"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// GroupResource ensures that the given group exists
type GroupResource struct {
	ResourceMeta
	Group string
	GID   uint32
}

// ShouldSkip tests that the group exists and has the correct name
func (gr *GroupResource) ShouldSkip(context.Context) (bool, error) {
	groupContents, err := ioutil.ReadFile("/etc/group")
	if err != nil {
		return false, errors.Wrap(err, "could not read /etc/group")
	}

	for i, line := range strings.Split(string(groupContents), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) != 4 {
			gr.Logger.Warnf("/etc/group malformed on line %v", i+1)
			continue
		}

		if parts[2] != strconv.Itoa(int(gr.GID)) {
			continue
		}
		if parts[0] != gr.Group {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// Materialize creates the group
func (gr *GroupResource) Materialize(context.Context) error {
	groupContents, err := ioutil.ReadFile("/etc/group")
	if err != nil {
		return errors.Wrap(err, "could not read /etc/group")
	}

	newContents := bytes.NewBuffer(nil)
	for i, line := range strings.Split(string(groupContents), "\n") {
		if len(line) == 0 {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 4 {
			gr.Logger.Warnf("/etc/group malformed on line %v", i+1)
			newContents.WriteString(line)
		} else if parts[2] != strconv.Itoa(int(gr.GID)) {
			newContents.WriteString(line)
		} else {
			parts[0] = gr.Name
			newContents.WriteString(strings.Join(parts, ":"))
		}
		newContents.WriteByte('\n')
	}
	err = ioutil.WriteFile("/etc/group", newContents.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write to /etc/group")
	}
	return nil
}

// GroupMembershipResource ensures that a user belongs to a group
type GroupMembershipResource struct {
	ResourceMeta
	GID  uint32
	User string
}

// ShouldSkip tests that the user belongs to the group
func (gmr *GroupMembershipResource) ShouldSkip(context.Context) (bool, error) {
	groupContents, err := ioutil.ReadFile("/etc/group")
	if err != nil {
		return false, errors.Wrap(err, "could not read /etc/group")
	}

	for i, line := range strings.Split(string(groupContents), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) != 4 {
			gmr.Logger.Warnf("/etc/group malformed on line %v", i+1)
			continue
		}

		if parts[2] != strconv.Itoa(int(gmr.GID)) {
			continue
		}
		members := strings.Split(parts[3], ",")
		for _, member := range members {
			if member == gmr.User {
				return true, nil
			}
		}
		return false, nil
	}
	return false, errors.Errorf("/etc/passwd did not contain group %v", gmr.GID)
}

// Materialize adds the user to the group
func (gmr *GroupMembershipResource) Materialize(context.Context) error {
	groupContents, err := ioutil.ReadFile("/etc/group")
	if err != nil {
		return errors.Wrap(err, "could not read /etc/group")
	}

	newContents := bytes.NewBuffer(nil)
	for i, line := range strings.Split(string(groupContents), "\n") {
		if len(line) == 0 {
			continue
		}
		newContents.WriteString(line)
		parts := strings.Split(line, ":")
		if len(parts) != 4 {
			gmr.Logger.Warnf("/etc/group malformed on line %v", i+1)
		} else if parts[2] == strconv.Itoa(int(gmr.GID)) {
			if parts[3] != "" {
				newContents.WriteByte(',')
			}
			newContents.WriteString(gmr.User)
		}
		newContents.WriteByte('\n')
	}
	err = ioutil.WriteFile("/etc/group", newContents.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write to /etc/group")
	}
	return nil
}
