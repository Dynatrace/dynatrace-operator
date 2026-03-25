package kspm

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
)

func (kspm *KSPM) SetName(name string) {
	kspm.name = name
}

func (kspm *KSPM) IsEnabled() bool {
	return kspm.Spec != nil
}

func (kspm *KSPM) GetTokenSecretName() string {
	return kspm.name + "-" + TokenSecretKey
}

func (kspm *KSPM) GetDaemonSetName() string {
	return kspm.name + consts.NodeCollectorNameSuffix
}

func (kspm *KSPM) GetUniqueMappedHostPaths() []string {
	tmpMappedHostPaths := append([]string{}, kspm.MappedHostPaths...)
	slices.Sort(tmpMappedHostPaths)

	return slices.Compact(tmpMappedHostPaths)
}
