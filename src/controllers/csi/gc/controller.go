package csigc

import (
	"context"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/pkg/errors"
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

// garbageCollectionInfo stores tenant specific information
// used to delete unused files or directories connected to that tenant
type garbageCollectionInfo struct {
	tenantUUID         string
	latestAgentVersion string
	pinnedVersions     pinnedVersionSet
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

	dynakube, err := getDynakubeFromRequest(ctx, gc.apiReader, request)
	if err != nil {
		return reconcileResult, err
	}
	if dynakube == nil {
		return reconcileResult, nil
	}

	if !isSafeToGC(gc.db){
		log.Info("dynakube metadata is in a unfinished state, checking later")
		return reconcileResult, nil
	}

	gcInfo, err := collectGCInfo(ctx, gc.apiReader, *dynakube)
	if err != nil {
		return reconcileResult, err
	}
	if gcInfo == nil {
		return reconcileResult, nil
	}

	log.Info("running binary garbage collection")
	gc.runBinaryGarbageCollection(gcInfo.pinnedVersions, gcInfo.tenantUUID, gcInfo.latestAgentVersion)

	log.Info("running log garbage collection")
	gc.runLogGarbageCollection(gcInfo.tenantUUID)

	log.Info("running shared images garbage collection")
	if err := gc.runSharedImagesGarbageCollection(); err != nil {
		log.Info("failed to garbage collect the shared images")
		return reconcileResult, err
	}

	return reconcileResult, nil
}

func getDynakubeFromRequest(ctx context.Context, apiReader client.Reader, request reconcile.Request) (*dynatracev1beta1.DynaKube, error) {
	var dynakube dynatracev1beta1.DynaKube
	if err := apiReader.Get(ctx, request.NamespacedName, &dynakube); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("given DynaKube object not found")
			return nil, nil
		}

		log.Info("failed to get DynaKube object")
		return nil, errors.WithStack(err)
	}
	return &dynakube, nil
}

func collectGCInfo(ctx context.Context, apiReader client.Reader, dynakube dynatracev1beta1.DynaKube) (*garbageCollectionInfo, error) {
	tenantUUID, err := dynakube.TenantUUID()
	if err != nil {
		log.Info("failed to get tenantUUID of DynaKube, checking later")
		return nil, nil
	}

	latestAgentVersion := dynakube.Status.LatestAgentVersionUnixPaas
	if latestAgentVersion == "" {
		log.Info("no latest agent version found in dynakube, checking later")
		return nil, nil
	}

	pinnedVersions, err := getAllPinnedVersionsForTenantUUID(ctx, apiReader, tenantUUID, dynakube.Namespace)
	if err != nil {
		log.Info("failed to determine pinned agent versions")
		return nil, err
	}

	return &garbageCollectionInfo{
		tenantUUID:         tenantUUID,
		latestAgentVersion: latestAgentVersion,
		pinnedVersions:     pinnedVersions,
	}, nil
}

func isSafeToGC(access metadata.Access) bool {
	dkMetadataList, err := access.GetAllDynakubes()
	if err != nil {
		log.Info("failed to get dynakube metadata from database, err: %s")
		return false
	}
	for _, dkMetadata := range dkMetadataList {
		if dkMetadata.LatestVersion == "" {
			return false
		}
	}
	return true
}

// getAllPinnedVersionsForTenantUUID returns all pinned versions for a given tenantUUID.
// A pinned version is either:
// - the image tag or digest set in the custom resource (this doesn't matter in context of the GC)
// - the version set in the custom resource if applicationMonitoring is used
func getAllPinnedVersionsForTenantUUID(ctx context.Context, apiReader client.Reader, tenantUUID, namespace string) (pinnedVersionSet, error) {
	dynakubeList, err := getAllDynakubes(ctx, apiReader, namespace)
	if err != nil {
		return nil, err
	}
	pinnedVersions := make(pinnedVersionSet)
	for _, dynakube := range dynakubeList.Items {
		uuid, err := dynakube.TenantUUID()
		if err != nil {
			log.Error(err, "failed to get tenantUUID of DynaKube")
			continue
		}
		if uuid != tenantUUID {
			continue
		}
		codeModuleVersion := dynakube.CodeModulesVersion()
		if codeModuleVersion != "" {
			pinnedVersions[codeModuleVersion] = true
		}
	}
	return pinnedVersions, nil
}

func getAllDynakubes(ctx context.Context, apiReader client.Reader, namespace string) (*dynatracev1beta1.DynaKubeList, error) {
	var dynakubeList dynatracev1beta1.DynaKubeList
	if err := apiReader.List(ctx, &dynakubeList, client.InNamespace(namespace)); err != nil {
		log.Info("failed to get all DynaKube objects")
		return nil, errors.WithStack(err)
	}
	return &dynakubeList, nil
}
