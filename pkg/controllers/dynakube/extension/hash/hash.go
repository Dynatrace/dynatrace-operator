package hash

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
)

func SetHash(o *appsv1.StatefulSet) error {
	hash, err := hasher.GenerateHash(o)
	if err != nil {
		return errors.WithStack(err)
	}

	o.ObjectMeta.Annotations[hasher.AnnotationHash] = hash

	return nil
}
