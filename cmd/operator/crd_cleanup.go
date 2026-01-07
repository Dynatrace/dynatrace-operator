package operator

import (
	"context"

	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func runCRDCleanup(cfg *rest.Config, namespace string) error {
	crdCleanupManager, err := createCRDCleanupManager(cfg, namespace)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	// Start the manager in a goroutine to make liveness probe available
	managerErrChan := make(chan error, 1)
	go func() {
		if err := crdCleanupManager.Start(ctx); err != nil {
			managerErrChan <- err
		}
	}()

	cleanupErr := performCRDStorageVersionsCleanup(ctx, crdCleanupManager.GetClient())
	cancelFn()

	select {
	case err := <-managerErrChan:
		if cleanupErr == nil {
			return err
		}
		return cleanupErr
	default:
		return cleanupErr
	}
}

func createCRDCleanupManager(cfg *rest.Config, namespace string) (manager.Manager, error) {
	crdCleanupManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		HealthProbeBindAddress: healthProbeBindAddress,
		LivenessEndpointName:   livenessEndpointName,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = crdCleanupManager.AddHealthzCheck(livezEndpointName, healthz.Ping)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return crdCleanupManager, nil
}

// performCRDStorageVersionsCleanup performs cleanup of CRD storage versions before operator startup.
// It checks if the DynaKube CRD has multiple storage versions and if so, reads and writes
// all DynaKube instances to migrate them to the current storage version.
// TODO gakr: Add working unit and integration tests for this function.
// TODO gakr: Include logic for EdgeConnect CRD
func performCRDStorageVersionsCleanup(ctx context.Context, clt client.Client) error {
	log.Info("starting CRD storage version cleanup")

	var crd apiextensionsv1.CustomResourceDefinition
	err := clt.Get(ctx, types.NamespacedName{Name: k8scrd.DynaKubeName}, &crd)
	if err != nil {
		log.Info("failed to get DynaKube CRD, skipping cleanup", "error", err)
		return nil
	}

	if len(crd.Status.StoredVersions) == 0 {
		log.Info("DynaKube CRD has no storage versions, skipping cleanup")
		return nil
	}

	if len(crd.Status.StoredVersions) == 1 && crd.Status.StoredVersions[0] == latest.GroupVersion.Version {
		log.Info("DynaKube CRD has single, up-to-date storage version, no cleanup needed",
			"storedVersions", crd.Status.StoredVersions)
		return nil
	}

	log.Info("DynaKube CRD has multiple storage versions, performing migration",
		"storedVersions", crd.Status.StoredVersions,
		"currentVersion", latest.GroupVersion.Version)

	// List all DynaKube instances
	var dynakubeList dynakube.DynaKubeList
	err = clt.List(ctx, &dynakubeList, &client.ListOptions{
		Namespace: k8senv.DefaultNamespace(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list DynaKube instances")
	}

	log.Info("migrating DynaKube instances to current storage version",
		"count", len(dynakubeList.Items),
		"targetVersion", latest.GroupVersion.Version)

	for i := range dynakubeList.Items {
		dk := &dynakubeList.Items[i]
		log.Info("migrating DynaKube instance",
			"name", dk.Name,
			"namespace", dk.Namespace)

		err = clt.Update(ctx, dk)
		if err != nil {
			return errors.Wrapf(err, "failed to update DynaKube %s/%s", dk.Namespace, dk.Name)
		}
	}

	// Remove the old storage versions from the CRD status
	crd.Status.StoredVersions = []string{latest.GroupVersion.Version}
	err = clt.Status().Update(ctx, &crd)
	if err != nil {
		return errors.Wrap(err, "failed to update DynaKube CRD status")
	}

	log.Info("successfully migrated all DynaKube instances to current storage version")

	return nil
}
