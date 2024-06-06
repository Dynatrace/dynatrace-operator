package csigc

import (
	"context"
	"os"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CSIGarbageCollector removes unused and outdated agent versions
type CSIGarbageCollector struct {
	apiReader client.Reader
	fs        afero.Fs
	db        metadata.Cleaner
	path      metadata.PathResolver

	maxUnmountedVolumeAge time.Duration
}

var _ reconcile.Reconciler = (*CSIGarbageCollector)(nil)

// NewCSIGarbageCollector returns a new CSIGarbageCollector
func NewCSIGarbageCollector(apiReader client.Reader, opts dtcsi.CSIOptions, db metadata.Cleaner) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		apiReader:             apiReader,
		fs:                    afero.NewOsFs(),
		db:                    db,
		path:                  metadata.PathResolver{RootDir: opts.RootDir},
		maxUnmountedVolumeAge: determineMaxUnmountedVolumeAge(os.Getenv(maxUnmountedCsiVolumeAgeEnv)),
	}
}

func (gc *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("running OneAgent garbage collection", "namespace", request.Namespace, "name", request.Name)

	log.Info("running binary garbage collection")
	gc.runBinaryGarbageCollection()

	if err := ctx.Err(); err != nil {
		return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
	}

	tenantConfigs, err := gc.db.ListDeletedTenantConfigs()

	if err != nil {
		return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
	}

	log.Info("running log garbage collection")

	for _, tenantConfig := range tenantConfigs {
		log.Info("cleaning up soft deleted tenant-config", "name", tenantConfig.Name)

		gc.runUnmountedVolumeGarbageCollection(tenantConfig.TenantUUID)

		osMounts, err := gc.db.ListDeletedOSMounts()
		if err != nil {
			continue
		}

		for _, osm := range osMounts {
			if osm.TenantConfigUID == tenantConfig.UID {
				gc.fs.RemoveAll(osm.Location)
				gc.db.PurgeOSMount(&osm)
			}
		}

		err = gc.db.PurgeTenantConfig(&tenantConfig)
		if err != nil {
			log.Info("failed to remove the soft deleted tenant-config entry, will try again", "name", tenantConfig.Name)

			return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, nil //nolint: nilerr
		}
	}

	if err := ctx.Err(); err != nil {
		return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
	}

	return reconcile.Result{RequeueAfter: dtcsi.LongRequeueDuration}, nil
}
