// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package mount

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var errNotSupported = errors.New("not supported on darwin")

type osMounter struct{}

func New() Mounter {
	return &osMounter{}
}

// Mount is not supported on darwin. Overlay and bind mounts are linux-only in production;
// darwin support exists only to allow the codebase to compile during development.
func (m *osMounter) Mount(source, target, fstype string, options []string) error {
	return errNotSupported
}

func (m *osMounter) Unmount(target string) error {
	return exec.Command("umount", target).Run() //nolint:gosec
}

func (m *osMounter) IsMountPoint(target string) (bool, error) {
	var targetStat unix.Stat_t
	if err := unix.Lstat(target, &targetStat); err != nil {
		return false, &os.PathError{Op: "lstat", Path: target, Err: err}
	}

	parent := filepath.Dir(target)

	var parentStat unix.Stat_t
	if err := unix.Lstat(parent, &parentStat); err != nil {
		return false, &os.PathError{Op: "lstat", Path: parent, Err: err}
	}

	return targetStat.Dev != parentStat.Dev, nil
}

func (m *osMounter) List() ([]MountPoint, error) {
	return nil, nil
}
