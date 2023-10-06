package parametermap

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
)

type argument struct {
	delimiter string
	value     any
	priority  int
}

const defaultPriority = 1

type Map struct {
	arguments      map[string]argument
	defaultOptions []Option
}

type Option func(a *argument)

func WithPriority(priority int) Option {
	return func(a *argument) {
		a.priority = priority
	}
}

func WithSeparator(separator string) Option {
	return func(a *argument) {
		a.delimiter = separator
	}
}

func NewMap(defaultOptions ...Option) *Map {
	m := &Map{
		arguments:      make(map[string]argument),
		defaultOptions: defaultOptions,
	}
	return m
}

func (m Map) Append(key string, value any, opts ...Option) {
	if len(key) == 0 {
		return
	}

	newArg := argument{
		value:    value,
		priority: defaultPriority,
	}

	for _, opt := range m.defaultOptions {
		opt(&newArg)
	}
	for _, opt := range opts {
		opt(&newArg)
	}

	key, _ = strings.CutSuffix(key, newArg.delimiter)

	if existingArg, exists := m.arguments[key]; !exists || newArg.priority > existingArg.priority {
		m.arguments[key] = newArg
	}
}

type ValueType interface {
	corev1.EnvVar | []corev1.EnvVar | map[string]any | []string
}

func Append[V ValueType](argMap *Map, value V, opts ...Option) {
	switch typedValue := any(value).(type) {
	case corev1.EnvVar:
		argMap.Append(typedValue.Name, typedValue, opts...)
	case []corev1.EnvVar:
		for _, vv := range typedValue {
			argMap.Append(vv.Name, vv, opts...)
		}
	case map[string]any:
		for k, vv := range typedValue {
			argMap.Append(k, vv, opts...)
		}
	case []string:
		for _, s := range typedValue {
			key, delim, value := ParseCommandLineArgument(s)

			if len(key) > 0 {
				opts = append(opts, WithSeparator(delim))
				argMap.Append(key, value, opts...)
			}
		}
	}
}

func (m Map) AsEnvVars() []corev1.EnvVar {
	keys := m.getSortedKeys()
	envVars := make([]corev1.EnvVar, 0, len(keys))
	for _, key := range keys {
		switch typedValue := m.arguments[key].value.(type) {
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
	return envVars
}

func (m Map) AsKeyValueStrings() []string {
	keys := m.getSortedKeys()
	valStrings := make([]string, 0)
	for _, key := range keys {
		val := m.arguments[key]
		valStrings = append(valStrings, fmt.Sprintf("%s%s%v", key, val.delimiter, val.value))
	}
	return valStrings
}

func (m Map) getSortedKeys() []string {
	// some unit tests rely on having the resulting env vars always being in the same order
	keys := make([]string, 0, len(m.arguments))
	for key := range m.arguments {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
