package rfsb

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// CmdResource execs the specified command when materialized. It will not invoke a shell.
//
// For the ability to set UID and GID, see sudo
type CmdResource struct {
	ResourceMeta

	Command     string
	Arguments   []string
	CWD         string
	Environment map[string]string
}

// Materialize runs the specified command
func (cr *CmdResource) Materialize(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, cr.Command, cr.Arguments...)
	cr.Logger().Infof("running %s %s", cr.Command, strings.Join(cr.Arguments, " "))
	for k, v := range cr.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if cr.CWD != "" {
		cmd.Dir = cr.CWD
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			stdout := strings.TrimSpace(stdout.String())
			if stdout != "" {
				cr.Logger().Errorf("command stdout:\n%s", stdout)
			}
			stderr := strings.TrimSpace(stderr.String())
			if stderr != "" {
				cr.Logger().Errorf("command stderr:\n%s", stderr)
			}
			return errors.Wrap(err, "command failed")
		}
		return errors.Wrap(err, "could not run command")
	}

	return nil
}
