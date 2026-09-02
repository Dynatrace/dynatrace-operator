// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package mount

// Mounter abstracts OS-specific mount syscalls for CSI volume management.
type Mounter interface {
	Mount(source, target, fstype string, options []string) error
	Unmount(target string) error
	IsMountPoint(target string) (bool, error)
	List() ([]MountPoint, error)
}

// MountPoint represents an active mount entry.
type MountPoint struct {
	Device string
	Path   string
	Type   string
	Opts   []string
}
