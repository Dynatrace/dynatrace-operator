package metadata

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Checks if the entries in the storage are actually valid
// Removes not valid entries
func CheckStorageCorrectness(cl client.Client, access Access, log logr.Logger) error {
	if err := checkVolumesCorrectness(cl, access, log); err != nil {
		return err
	}
	if err := checkTenantsCorrectness(cl, access, log); err != nil {
		return err
	}
	return nil
}

// Removes volume entries if their pod is no longer exists
func checkVolumesCorrectness(cl client.Client, access Access, log logr.Logger) error {
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
	log.Info("CSI volumes database is corrected", "prunedRows", pruned)
	return nil
}

// Removes tenant entries if their dynakube no longer exists
func checkTenantsCorrectness(cl client.Client, access Access, log logr.Logger) error {
	dynakubes, err := access.GetDynakubes()
	if err != nil {
		return err
	}
	pruned := 0
	for dynakubeName := range dynakubes {
		var dynakube dynatracev1alpha1.DynaKube
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: dynakubeName}, &dynakube); !k8serrors.IsNotFound(err) {
			continue
		}
		if err := access.DeleteTenant(dynakubes[dynakubeName]); err != nil {
			return err
		}
		pruned++
	}
	log.Info("CSI tenants database is corrected", "prunedRows", pruned)
	return nil
}
