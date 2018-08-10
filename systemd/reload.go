// Package systemd implements some resources that make dealing with systemd a little easier. These functions and
// resources aren't anything you couldn't write yourself, but they're a bit of a pain to get right.
package systemd

import (
	"context"
	"os"
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/lclarkmichalek/rfsb"
	"github.com/pkg/errors"
)

// ReloadIfModified reloads the systemd daemon's config via `systemctl daemon-reload` if the unit file at the passed
// path has not been modified since the time specified by the beforeModification argument
func ReloadIfModified(unitPath string, beforeModification time.Time) rfsb.Resource {
	return &rfsb.SkippingWrapper{
		Resource: &DaemonReload{},
		SkipFunc: func(context.Context) (bool, error) {
			stat, err := os.Stat(unitPath)
			if err != nil {
				return false, errors.Wrap(err, "could not stat unit file")
			}
			if stat.ModTime().Before(beforeModification) {
				return true, nil
			}
			return false, nil
		},
	}
}

// DaemonReload triggers a reload of the systemd daemon when materialized
type DaemonReload struct{ rfsb.ResourceMeta }

// Materialize connects to the system dbus connection and triggers a reload
func (*DaemonReload) Materialize(context.Context) error {
	conn, err := dbus.New()
	if err != nil {
		return errors.Wrap(err, "could not create dbus connection")
	}
	defer conn.Close()

	err = conn.Reload()
	return errors.Wrap(err, "could not call daemon reload")
}

type StartUnit struct {
	rfsb.ResourceMeta

	UnitName string
}

func (su *StartUnit) Materialize(context.Context) error {
	conn, err := dbus.New()
	if err != nil {
		return errors.Wrap(err, "could not create dbus connection")
	}
	defer conn.Close()

	_, err = conn.StartUnit(su.UnitName, "replace", nil)
	return errors.Wrap(err, "could not start unit")
}
