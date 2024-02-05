package kubesystem

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Namespace             = "kube-system"
	olmSpecificAnnotation = "olm.operatorNamespace"
)

func GetUID(ctx context.Context, clt client.Reader) (types.UID, error) {
	ctx, span := dtotel.StartSpan(ctx, tracer())
	defer span.End()

	kubeSystemNamespace := &corev1.Namespace{}

	err := clt.Get(ctx, client.ObjectKey{Name: Namespace}, kubeSystemNamespace)
	if err != nil {
		span.RecordError(err)
		return "", errors.WithStack(err)
	}

	return kubeSystemNamespace.UID, nil
}

func IsDeployedViaOlm(pod corev1.Pod) bool {
	_, isDeployedViaOlm := pod.Annotations[olmSpecificAnnotation]
	return isDeployedViaOlm
}
