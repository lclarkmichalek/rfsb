package rfsb

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// UserResource ensures that the given user exists
type UserResource struct {
	ResourceMeta
	User  string
	UID   uint32
	GID   uint32
	Home  string
	Shell string
}

// passwdLine returns the line we would expect to see in /etc/passwd for this user
func (ur *UserResource) passwdLine() string {
	return fmt.Sprintf("%s:x:%d:%d::%s:%s", ur.User, ur.UID, ur.GID, ur.Home, ur.Shell)
}

func lineDefinesUID(line string, uid uint32) bool {
	parts := strings.Split(line, ":")
	if len(parts) != 7 {
		return false
	}
	return parts[3] == strconv.Itoa(int(uid))
}

// ShouldSkip tests that the user exists, and has the correct properties. If it does, the resource is already materialized and will not be rerun
func (ur *UserResource) ShouldSkip(context.Context) (bool, error) {
	passwdContents, err := ioutil.ReadFile("/etc/passwd")
	if err != nil {
		return false, errors.Wrap(err, "could not read /etc/passwd")
	}
	expectedLine := ur.passwdLine()
	for _, line := range strings.Split(string(passwdContents), "\n") {
		if line == expectedLine {
			return true, nil
		} else if lineDefinesUID(line, ur.UID) {
			ur.Logger().Infof("found user registered with different attributes")
			return false, nil
		}
	}
	return false, nil
}

// Materialize creates the user
func (ur *UserResource) Materialize(context.Context, chan<- Signal) error {
	passwdContents, err := ioutil.ReadFile("/etc/passwd")
	if err != nil {
		return errors.Wrap(err, "could not read /etc/passwd")
	}
	expectedLine := ur.passwdLine()
	newPasswd := bytes.NewBuffer(nil)
	for _, line := range strings.Split(string(passwdContents), "\n") {
		if len(line) == 0 {
			continue
		}
		if line == expectedLine {
			ur.Logger().Warnf("skip failed, user existed")
		} else if lineDefinesUID(line, ur.UID) {
			newPasswd.Write([]byte(expectedLine))
		} else {
			newPasswd.Write([]byte(line))
		}
		newPasswd.Write([]byte{'\n'})
	}

	err = ioutil.WriteFile("/etc/passwd", newPasswd.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write to /etc/passwd")
	}
	return nil
}
