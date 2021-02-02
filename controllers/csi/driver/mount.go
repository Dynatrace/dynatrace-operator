package csidriver

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"k8s.io/utils/mount"
)

type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

func BindMount(rootDir string, mnts ...Mount) error {
	if err := os.MkdirAll(rootDir, 0750); err != nil {
		return fmt.Errorf("failed to create root directory %s: %w", rootDir, err)
	}

	mounter := mount.New("")
	for i, mnt := range mnts {
		opts := []string{"bind"}
		if mnt.ReadOnly {
			opts = append(opts, "ro")
		}

		if err := mounter.Mount(mnt.Source, mnt.Target, "", opts); err != nil {
			var errList strings.Builder
			errList.WriteString(fmt.Sprintf("failed to mount device: %s at %s: %s", mnt.Source, mnt.Target, err.Error()))

			if err := BindUnmount(mnts[0:i]...); err != nil {
				errList.WriteString(fmt.Sprintf(", failed to unmount after failed operation: %s", err.Error()))
			} else {
				if err := os.RemoveAll(rootDir); err != nil {
					errList.WriteString(fmt.Sprintf(", failed to to delete root directory %s: %s", rootDir, err.Error()))
				}
			}

			return errors.New(errList.String())
		}
	}

	return nil
}

func BindUnmount(mnts ...Mount) error {
	var errList []string

	mounter := mount.New("")
	for _, mnt := range mnts {
		// Unmount only if the target path is really a mount point.
		notMnt, err := mount.IsNotMountPoint(mounter, mnt.Target)
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			errList = append(errList, fmt.Sprintf("failed to query if mount point on %s: %s", mnt.Target, err.Error()))
			continue
		}

		if notMnt {
			continue
		}

		// Unmounting the image or filesystem.
		if err := mounter.Unmount(mnt.Target); err != nil {
			errList = append(errList, fmt.Sprintf("failed to unmount %s: %s", mnt.Target, err.Error()))
			continue
		}
	}

	if len(errList) == 0 {
		return nil
	}

	return errors.New(strings.Join(errList, ","))
}
