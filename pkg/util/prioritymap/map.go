package prioritymap

import (
	"strings"

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
)

const DefaultPriority = LowPriority
const LowPriority = 1
const MediumPriority = 5
const HighPriority = 10

type Map struct {
	entries        map[string][]entry
	defaultOptions []Option
}

type entry struct {
	value           any
	key             string
	delimiter       string
	priority        int
	allowDuplicates bool
}

type Option func(a *entry)

func WithPriority(priority int) Option {
	return func(a *entry) {
		a.priority = priority
	}
}

func WithSeparator(separator string) Option {
	return func(a *entry) {
		a.delimiter = separator
	}
}

func WithAllowDuplicates() Option {
	return func(a *entry) {
		a.allowDuplicates = true
	}
}

func WithAvoidDuplicates() Option {
	return func(a *entry) {
		a.allowDuplicates = false
	}
}

func WithAvoidDuplicatesFor(key string) Option {
	return func(a *entry) {
		if a.key == key {
			a.allowDuplicates = false
		}
	}
}

func WithAllowDuplicatesFor(key string) Option {
	return func(a *entry) {
		if a.key == key {
			a.allowDuplicates = true
		}
	}
}

func New(defaultOptions ...Option) *Map {
	m := &Map{
		entries:        make(map[string][]entry),
		defaultOptions: defaultOptions,
	}

	return m
}

func (m Map) Append(key string, value any, opts ...Option) {
	if len(key) == 0 {
		return
	}

	newArg := entry{
		key:             key,
		value:           value,
		priority:        DefaultPriority,
		allowDuplicates: false,
	}

	for _, opt := range m.defaultOptions {
		opt(&newArg)
	}

	for _, opt := range opts {
		opt(&newArg)
	}

	key, _ = strings.CutSuffix(key, newArg.delimiter)

	if existingArg, exists := m.entries[key]; !exists || newArg.allowDuplicates || newArg.priority > existingArg[0].priority {
		if !exists || !newArg.allowDuplicates {
			m.entries[key] = make([]entry, 0)
		}

		if !slices.ContainsFunc(m.entries[key], func(e entry) bool {
			return e.value == value
		}) {
			m.entries[key] = append(m.entries[key], newArg)
		}
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
