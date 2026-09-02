// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package mount

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

type osMounter struct{}

func New() Mounter {
	return &osMounter{}
}

func (m *osMounter) Mount(source, target, fstype string, options []string) error {
	var flags uintptr

	var data []string

	for _, opt := range options {
		switch opt {
		case "bind":
			flags |= unix.MS_BIND
		case "ro":
			flags |= unix.MS_RDONLY
		case "remount":
			flags |= unix.MS_REMOUNT
		default:
			data = append(data, opt)
		}
	}

	return unix.Mount(source, target, fstype, flags, strings.Join(data, ","))
}

func (m *osMounter) Unmount(target string) error {
	return unix.Unmount(target, 0)
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
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil, err
	}

	var mounts []MountPoint

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		mounts = append(mounts, MountPoint{
			Device: fields[0],
			Path:   fields[1],
			Type:   fields[2],
			Opts:   strings.Split(fields[3], ","),
		})
	}

	return mounts, nil
}
