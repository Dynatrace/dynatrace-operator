package prioritymap

import (
	"strings"

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
