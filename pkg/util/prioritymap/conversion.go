// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prioritymap

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	corev1 "k8s.io/api/core/v1"
)

func (m Map) AsEnvVars() []corev1.EnvVar {
	keys := slices.Sorted(maps.Keys(m.entries))
	envVars := make([]corev1.EnvVar, 0, len(keys))

	for _, key := range keys {
		for _, entry := range m.entries[key] {
			switch typedValue := entry.value.(type) {
			case string:
				envVars = append(envVars, corev1.EnvVar{
					Name:  key,
					Value: typedValue,
				})
			case corev1.EnvVar:
				envVars = append(envVars, typedValue)
			case *corev1.EnvVar:
				envVars = append(envVars, *typedValue)
			case *corev1.EnvVarSource:
				envVars = append(envVars, corev1.EnvVar{
					Name:      key,
					ValueFrom: typedValue,
				})
			case corev1.EnvVarSource:
				envVars = append(envVars, corev1.EnvVar{
					Name:      key,
					ValueFrom: &typedValue,
				})
			}
		}
	}

	return envVars
}

func (m Map) AsKeyValueStrings() []string {
	keys := slices.Sorted(maps.Keys(m.entries))
	valStrings := make([]string, 0, len(keys))

	for _, key := range keys {
		slices.SortStableFunc(m.entries[key], func(a, b entry) int {
			return cmp.Compare(a.priority, b.priority)
		})

		for _, entry := range m.entries[key] {
			valStrings = append(valStrings, fmt.Sprintf("%s%s%v", key, entry.delimiter, entry.value))
		}
	}

	return valStrings
}
