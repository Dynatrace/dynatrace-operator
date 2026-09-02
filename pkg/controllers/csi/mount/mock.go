// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package mount

// MockMounter is a test double for Mounter.
// MountCheckErrors controls per-path IsMountPoint behavior: nil error → is a mount point, non-nil → return that error.
type MockMounter struct {
	MountPoints      []MountPoint
	MountCheckErrors map[string]error
}

func NewMockMounter(mountPoints ...MountPoint) *MockMounter {
	return &MockMounter{
		MountPoints: mountPoints,
	}
}

func (m *MockMounter) Mount(source, target, fstype string, options []string) error {
	m.MountPoints = append(m.MountPoints, MountPoint{
		Device: source,
		Path:   target,
		Type:   fstype,
		Opts:   options,
	})

	return nil
}

func (m *MockMounter) Unmount(target string) error {
	for i, mp := range m.MountPoints {
		if mp.Path == target {
			m.MountPoints = append(m.MountPoints[:i], m.MountPoints[i+1:]...)

			return nil
		}
	}

	return nil
}

func (m *MockMounter) IsMountPoint(target string) (bool, error) {
	if m.MountCheckErrors != nil {
		err, ok := m.MountCheckErrors[target]
		if ok {
			if err == nil {
				return true, nil
			}

			return false, err
		}
	}

	for _, mp := range m.MountPoints {
		if mp.Path == target {
			return true, nil
		}
	}

	return false, nil
}

func (m *MockMounter) List() ([]MountPoint, error) {
	return m.MountPoints, nil
}
