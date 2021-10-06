package metadata

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CorrectMetadata checks if the entries in the storage are actually valid
// Removes not valid entries
func CorrectMetadata(cl client.Client, access Access, log logr.Logger) error {
	if err := correctVolumes(cl, access, log); err != nil {
		return err
	}
	if err := correctTenants(cl, access, log); err != nil {
		return err
	}
	return nil
}

// Removes volume entries if their pod is no longer exists
func correctVolumes(cl client.Client, access Access, log logr.Logger) error {
	podNames, err := access.GetPodNames()
	if err != nil {
		return err
	}
	pruned := []string{}
	for podName := range podNames {
		var pod corev1.Pod
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: podName}, &pod); !k8serrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return err
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
func correctTenants(cl client.Client, access Access, log logr.Logger) error {
	tenants, err := access.GetTenants()
	if err != nil {
		return err
	}
	pruned := []string{}
	for dynakubeName := range tenants {
		var dynakube dynatracev1beta1.DynaKube
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: dynakubeName}, &dynakube); !k8serrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := access.DeleteTenant(dynakubeName); err != nil {
			return err
		}
		tenantUUID := tenants[dynakubeName]
		pruned = append(pruned, tenantUUID+"|"+dynakubeName)
	}
	log.Info("CSI tenants database is corrected (tenant|dynakube)", "prunedRows", pruned)
	return nil
}
