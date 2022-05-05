package csigc

import (
	"context"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/spf13/afero"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// can contain the tag of the image or the digest, depending on how the user provided the image
// or the version set for the download
type pinnedVersionSet map[string]bool

func (set pinnedVersionSet) isNotPinned(version string) bool {
	return !set[version]
}

// CSIGarbageCollector removes unused and outdated agent versions
type CSIGarbageCollector struct {
	apiReader client.Reader
	opts      dtcsi.CSIOptions
	fs        afero.Fs
	db        metadata.Access
	path      metadata.PathResolver
}

// NewCSIGarbageCollector returns a new CSIGarbageCollector
func NewCSIGarbageCollector(apiReader client.Reader, opts dtcsi.CSIOptions, db metadata.Access) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		apiReader: apiReader,
		opts:      opts,
		fs:        afero.NewOsFs(),
		db:        db,
		path:      metadata.PathResolver{RootDir: opts.RootDir},
	}
}

func (gc *CSIGarbageCollector) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Complete(gc)
}

func (gc *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("running OneAgent garbage collection", "namespace", request.Namespace, "name", request.Name)
	reconcileResult := reconcile.Result{RequeueAfter: 60 * time.Minute}

	var dynakube dynatracev1beta1.DynaKube
	if err := gc.apiReader.Get(ctx, request.NamespacedName, &dynakube); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("given DynaKube object not found")
			return reconcileResult, nil
		}

		log.Error(err, "failed to get DynaKube object")
		return reconcileResult, nil
	}

	tenantUUID, err := dynakube.TenantUUID()
	if err != nil {
		log.Error(err, "failed to get tenantUUID of DynaKube")
		return reconcileResult, err
	}

	latestAgentVersion := dynakube.Status.LatestAgentVersionUnixPaas
	if latestAgentVersion == "" {
		log.Info("no latest agent version found in dynakube, checking later")
		return reconcileResult, nil
	}

	var dynakubeList dynatracev1beta1.DynaKubeList
	if err := gc.apiReader.List(ctx, &dynakubeList, client.InNamespace(dynakube.Namespace)); err != nil {
		log.Error(err, "failed to get all DynaKube objects")
		return reconcileResult, err
	}

	pinnedVersions := getAllPinnedVersionsForTenantUUID(tenantUUID, dynakubeList)
	log.Info("running binary garbage collection")
	gc.runBinaryGarbageCollection(pinnedVersions, tenantUUID, latestAgentVersion)

	log.Info("running log garbage collection")
	gc.runLogGarbageCollection(tenantUUID)

	log.Info("running image garbage collection")
	gc.runImageGarbageCollection()

	return reconcileResult, nil
}

// getAllPinnedVersionsForTenantUUID returns all pinned versions for a given tenantUUID.
// A pinned version is either:
// - the image tag or digest set in the custom resource
// - the version set in the custom resource if applicationMonitoring is used
func getAllPinnedVersionsForTenantUUID(tenantUUID string, dynakubes dynatracev1beta1.DynaKubeList) pinnedVersionSet {
	pinnedImages := make(pinnedVersionSet)
	for _, dynakube := range dynakubes.Items {
		uuid, err := dynakube.TenantUUID()
		if err != nil {
			log.Error(err, "failed to get tenantUUID of DynaKube")
			continue
		}
		if uuid != tenantUUID {
			continue
		}
		pinnedImages[dynakube.CodeModulesVersion()] = true
	}
	return pinnedImages
}
