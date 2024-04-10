package prioritymap

import (
	"fmt"
	"sort"

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
)

func (m Map) AsEnvVars() []corev1.EnvVar {
	keys := m.getSortedKeys()
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
	keys := m.getSortedKeys()
	valStrings := make([]string, 0)

	for _, key := range keys {
		sort.SliceStable(m.entries[key], func(i, j int) bool {
			return m.entries[key][i].priority < m.entries[key][j].priority
		})

		for _, entry := range m.entries[key] {
			valStrings = append(valStrings, fmt.Sprintf("%s%s%v", key, entry.delimiter, entry.value))
		}
	}

	return valStrings
}

func (m Map) getSortedKeys() []string {
	// some unit tests rely on having the resulting env vars always being in the same order
	keys := make([]string, 0, len(m.entries))
	for key := range m.entries {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}
