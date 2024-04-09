package prioritymap

import (
	"fmt"
	"strings"

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
	delimiter       string
	priority        int
	allowDuplicates bool
}

type Option func(key string, a *entry)

func WithPriority(priority int) Option {
	return func(_ string, a *entry) {
		a.priority = priority
	}
}

func WithSeparator(separator string) Option {
	return func(_ string, a *entry) {
		a.delimiter = separator
	}
}

// WithAllowDuplicatesForKey allows to add multiple values for the same key (covers all keys)
func WithAllowDuplicates() Option {
	return func(_ string, a *entry) {
		a.allowDuplicates = true
	}
}

// WithAvoidDuplicates makes sure that only the last value added per key is kept in the map (covers all keys in map).
func WithAvoidDuplicates() Option {
	return func(_ string, a *entry) {
		a.allowDuplicates = false
	}
}

// WithAllowDuplicatesForKey allows to add multiple values for the same key
func WithAllowDuplicatesForKey(allowedKey string) Option {
	return func(key string, a *entry) {
		// at this point key could still have a pre- or postfix
		if strings.Contains(key, allowedKey) {
			a.allowDuplicates = true
		}
	}
}

// WithAvoidDuplicatesForKey makes sure that only the last value added for a given key is kept in the map
func WithAvoidDuplicatesForKey(allowedKey string) Option {
	return func(key string, a *entry) {
		// at this point key could still have a pre- or postfix
		if strings.Contains(key, allowedKey) {
			a.allowDuplicates = false
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
		value:           value,
		priority:        DefaultPriority,
		allowDuplicates: false,
	}

	for _, opt := range m.defaultOptions {
		opt(key, &newArg)
	}

	for _, opt := range opts {
		opt(key, &newArg)
	}

	key, _ = strings.CutSuffix(key, newArg.delimiter)

	if existingArg, exists := m.entries[key]; !exists || newArg.allowDuplicates || newArg.priority > existingArg[0].priority {
		if !exists || !newArg.allowDuplicates {
			m.entries[key] = make([]entry, 0)
		}

		if !newArg.allowDuplicates {
			log.Info(fmt.Sprintf("value for %s replaced by %s", key, newArg.value))
		}

		m.entries[key] = append(m.entries[key], newArg)
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
