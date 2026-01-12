package crdstoragemigration

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DynaKubeCRDName is the full name of the DynaKube CRD
	DynaKubeCRDName = k8scrd.DynaKubeName
)

func PerformCRDStorageVersionMigration(ctx context.Context, clt client.Client, apiReader client.Reader, namespace string) (bool, error) {
	log.Info("starting CRD storage version cleanup")

	var crd apiextensionsv1.CustomResourceDefinition

	err := apiReader.Get(ctx, types.NamespacedName{Name: DynaKubeCRDName}, &crd)
	if err != nil {
		log.Info("failed to get DynaKube CRD, skipping cleanup", "error", err)

		return false, nil
	}

	if len(crd.Status.StoredVersions) == 0 {
		log.Info("DynaKube CRD has no storage versions, skipping cleanup")

		return false, nil
	}

	targetVersion := GetLatestStorageVersion(&crd)
	if targetVersion == "" {
		return false, errors.New("failed to determine target storage version")
	}

	if len(crd.Status.StoredVersions) == 1 && crd.Status.StoredVersions[0] == targetVersion {
		log.Info("DynaKube CRD has single, up-to-date storage version, no cleanup needed",
			"storedVersions", crd.Status.StoredVersions)

		return false, nil
	}

	log.Info("DynaKube CRD has multiple storage versions, performing migration",
		"storedVersions", crd.Status.StoredVersions,
		"targetVersion", targetVersion)

	// List all DynaKube instances using unstructured to avoid version conflicts
	dynakubeList := &unstructured.UnstructuredList{}
	dynakubeList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "dynatrace.com",
		Version: targetVersion,
		Kind:    "DynaKubeList",
	})

	err = apiReader.List(ctx, dynakubeList, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list DynaKube instances")
	}

	log.Info("migrating DynaKube instances to current storage version",
		"count", len(dynakubeList.Items),
		"targetVersion", targetVersion)

	for i := range dynakubeList.Items {
		dk := &dynakubeList.Items[i]
		log.Info("migrating DynaKube instance",
			"name", dk.GetName(),
			"namespace", dk.GetNamespace())

		err = clt.Update(ctx, dk)
		if err != nil {
			return false, errors.Wrapf(err, "failed to update DynaKube %s/%s", dk.GetNamespace(), dk.GetName())
		}
	}

	crd.Status.StoredVersions = []string{targetVersion}

	err = clt.Status().Update(ctx, &crd)
	if err != nil {
		return false, errors.Wrap(err, "failed to update DynaKube CRD status")
	}

	log.Info("successfully migrated all DynaKube instances to current storage version")

	return true, nil
}

func GetLatestStorageVersion(crd *apiextensionsv1.CustomResourceDefinition) string {
	for _, version := range crd.Spec.Versions {
		if version.Storage {
			return version.Name
		}
	}

	return ""
}
