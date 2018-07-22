package rfsb

import (
	"context"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// FileResource ensures the file at the given path has the given content, mode and owner.
//
// It does not create directories. For that, see DirectoryResource
type FileResource struct {
	ResourceMeta
	Path     string
	Mode     os.FileMode
	UID      uint32
	GID      uint32
	Contents string
}

// ShouldSkip stats and reads the file to see if any modifications are required.
func (fr *FileResource) ShouldSkip(context.Context) (bool, error) {
	fi, err := os.Stat(fr.Path)
	if err != nil {
		if os.IsNotExist(err) {
			fr.Logger().Debugf("target does not exist")
			return true, nil
		}
		return false, errors.Wrap(err, "could not stat file")
	}
	if fi.Mode() != fr.Mode {
		fr.Logger().Infof("mode has changed (current: %v)", fr.Mode)
		return false, nil
	}
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		if sys.Uid != fr.UID || sys.Gid != fr.GID {
			fr.Logger().Infof("uid/gid has changed (current: %v:%v)", sys.Uid, sys.Gid)
			return false, nil
		}
	} else {
		fr.Logger().Warn("could not test file permissions as not linux")
	}

	currentContents, err := ioutil.ReadFile(fr.Path)
	if err != nil {
		return false, errors.Wrapf(err, "could not read %v", fr.Path)
	}

	return string(currentContents) == fr.Contents, nil
}

// Materialize writes the file out and sets the owners correctly
func (fr *FileResource) Materialize(context.Context) error {
	err := ioutil.WriteFile(fr.Path, []byte(fr.Contents), fr.Mode)
	if err != nil {
		return errors.Wrapf(err, "could not write to %v", fr.Path)
	}
	err = os.Chown(fr.Path, int(fr.UID), int(fr.GID))
	if err != nil {
		return errors.Wrapf(err, "could not set owner of %v", fr.Path)
	}
	return nil
}
