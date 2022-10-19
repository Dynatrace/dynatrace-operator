package metadata

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CorrectMetadata checks if the entries in the storage are actually valid
// Removes not valid entries
func CorrectMetadata(ctx context.Context, cl client.Client, access Access) error {
	defer LogAccessOverview(access)
	if err := correctVolumes(ctx, cl, access); err != nil {
		return err
	}
	if err := correctDynakubes(ctx, cl, access); err != nil {
		return err
	}
	return nil
}

// Removes volume entries if their pod is no longer exists
func correctVolumes(ctx context.Context, cl client.Client, access Access) error {
	podNames, err := access.GetPodNames(ctx)
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
		if err := access.DeleteVolume(ctx, volumeID); err != nil {
			return err
		}
		pruned = append(pruned, volumeID+"|"+podName)
	}
	log.Info("CSI volumes database is corrected (volume|pod)", "prunedRows", pruned)
	return nil
}

// Removes dynakube entries if their Dynakube instance no longer exists in the cluster
func correctDynakubes(ctx context.Context, cl client.Client, access Access) error {
	dynakubes, err := access.GetTenantsToDynakubes(ctx)
	if err != nil {
		return err
	}
	pruned := []string{}
	for dynakubeName := range dynakubes {
		var dynakube dynatracev1beta1.DynaKube
		if err := cl.Get(context.TODO(), client.ObjectKey{Name: dynakubeName}, &dynakube); !k8serrors.IsNotFound(err) {
			continue
		}
		if err := access.DeleteDynakube(ctx, dynakubeName); err != nil {
			return err
		}
		tenantUUID := dynakubes[dynakubeName]
		pruned = append(pruned, tenantUUID+"|"+dynakubeName)
	}
	log.Info("CSI tenants database is corrected (tenant|dynakube)", "prunedRows", pruned)
	return nil
}
