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
	pruned := []string{}
	for podName := range podNames {
		var pod corev1.Pod
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: podName}, &pod); !k8serrors.IsNotFound(err) {
			continue
		}
		volumeID := podNames[podName]
		if err := access.DeleteVolume(volumeID); err != nil {
			return err
		}
		pruned = append(pruned, volumeID+"|"+podName)
	}
	log.Info("CSI volumes database is corrected (volume|pod)", "prunedRows", pruned)
	return nil
}

// Removes tenant entries if their dynakube no longer exists
func checkTenantsCorrectness(cl client.Client, access Access, log logr.Logger) error {
	dynakubes, err := access.GetDynakubes()
	if err != nil {
		return err
	}
	pruned := []string{}
	for dynakubeName := range dynakubes {
		var dynakube dynatracev1alpha1.DynaKube
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: dynakubeName}, &dynakube); !k8serrors.IsNotFound(err) {
			continue
		}
		tenantUUID := dynakubes[dynakubeName]
		if err := access.DeleteTenant(tenantUUID); err != nil {
			return err
		}
		pruned = append(pruned, tenantUUID+"|"+dynakubeName)
	}
	log.Info("CSI tenants database is corrected (tenant|dynakube)", "prunedRows", pruned)
	return nil
}
