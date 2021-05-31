/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

type bindOptions struct {
	rootDir string
	mounts  []Mount
	mounter mount.Interface
}

func BindMount(rootDir string, mnts ...Mount) error {
	if err := os.MkdirAll(rootDir, 0750); err != nil {
		return fmt.Errorf("failed to create root directory %s: %w", rootDir, err)
	}

	return bindMount(&bindOptions{
		mounter: mount.New(""),
		mounts:  mnts,
		rootDir: rootDir,
	})
}

func bindMount(options *bindOptions) error {
	mounter := options.mounter
	mnts := options.mounts
	rootDir := options.rootDir

	for i, mnt := range mnts {
		opts := []string{"bind"}
		if mnt.ReadOnly {
			opts = append(opts, "ro")
		}

		if err := mounter.Mount(mnt.Source, mnt.Target, "", opts); err != nil {
			var errList strings.Builder
			errList.WriteString(fmt.Sprintf("failed to mount device: %s at %s: %s", mnt.Source, mnt.Target, err.Error()))

			if err := bindUnmount(&bindOptions{
				mounter: mounter,
				mounts:  mnts[0:i],
				rootDir: rootDir,
			}); err != nil {
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
	mounter := mount.New("")
	return bindUnmount(&bindOptions{
		mounter: mounter,
		mounts:  mnts,
	})
}

func bindUnmount(options *bindOptions) error {
	var errList []string
	mounter := options.mounter
	mnts := options.mounts

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
