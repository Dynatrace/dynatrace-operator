package builder

import (
	appsv1 "k8s.io/api/apps/v1"
)

type Modifier interface {
	Enabled() bool
	Modify(*appsv1.StatefulSet) error
}

type Builder struct {
	data      *appsv1.StatefulSet
	modifiers []Modifier
}

func NewBuilder(data appsv1.StatefulSet) Builder {
	return Builder{
		data:      &data,
		modifiers: []Modifier{},
	}
}

func (b *Builder) AddModifier(modifiers ...Modifier) *Builder {
	b.modifiers = append(b.modifiers, modifiers...)

	return b
}

func (b Builder) Build() (appsv1.StatefulSet, error) {
	var data appsv1.StatefulSet
	if b.data == nil {
		b.data = &data
	}

	for _, m := range b.modifiers {
		if m.Enabled() {
			err := m.Modify(b.data)
			if err != nil {
				return *b.data, err
			}
		}
	}

	return *b.data, nil
}
