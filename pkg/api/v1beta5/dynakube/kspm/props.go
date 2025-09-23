package kspm

import (
	"slices"
)

func (kspm *Kspm) SetName(name string) {
	kspm.name = name
}

func (kspm *Kspm) IsEnabled() bool {
	return kspm.Spec != nil
}

func (kspm *Kspm) GetTokenSecretName() string {
	return kspm.name + "-" + TokenSecretKey
}

func (kspm *Kspm) GetDaemonSetName() string {
	return kspm.name + "-" + NodeCollectorNameSuffix
}

func (kspm *Kspm) GetUniqueMappedHostPaths() []string {
	tmpMappedHostPaths := append([]string{}, kspm.MappedHostPaths...)
	slices.Sort(tmpMappedHostPaths)

	return slices.Compact(tmpMappedHostPaths)
}
