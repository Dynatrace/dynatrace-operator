package csigc

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/controllers/csi/storage"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckStorageCorrectness(cl client.Client, access storage.Access, log logr.Logger) error {
	if err := access.Setup(); err != nil {
		return err
	}
	podNames, err := access.GetPodNames()
	if err != nil {
		return err
	}
	pruned := 0
	for podName := range podNames {
		var pod corev1.Pod
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: podName}, &pod); !k8serrors.IsNotFound(err) {
			continue
		}
		if err := access.DeleteVolumeInfo(podNames[podName]); err != nil {
			return err
		}
		pruned++
	}
	log.Info("CSI database is set up and correct", "prunedRows", pruned)
	return nil
}
