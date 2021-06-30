package utils

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/statefulset"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdateDaemonSet(c client.Client, logger logr.Logger, desiredDs *appsv1.DaemonSet) (bool, error) {
	currentDs, err := getDaemonSet(c, desiredDs)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		logger.Info("creating new daemonset set for CSI driver")
		return true, c.Create(context.TODO(), desiredDs)
	} else if err != nil {
		return false, nil
	}

	if !HasDaemonSetChanged(currentDs, desiredDs) {
		return false, nil
	}

	logger.Info("updating existing CSI driver daemonset")
	if err = c.Update(context.TODO(), desiredDs); err != nil {
		return false, err
	}
	return true, err
}

func getDaemonSet(c client.Client, desiredDs *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	var actualDs appsv1.DaemonSet
	err := c.Get(context.TODO(), client.ObjectKey{Name: desiredDs.Name, Namespace: desiredDs.Namespace}, &actualDs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &actualDs, nil
}

func GenerateDaemonSetHash(ds *appsv1.DaemonSet) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func HasDaemonSetChanged(a, b *appsv1.DaemonSet) bool {
	return getTemplateHash(a) != getTemplateHash(b)
}

func getTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[statefulset.AnnotationTemplateHash]
	}
	return ""
}
