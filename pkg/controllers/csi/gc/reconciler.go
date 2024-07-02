package csigc

import (
	"context"
	"os"
	"path"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/spf13/afero"
	"k8s.io/utils/mount"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CSIGarbageCollector removes unused and outdated agent versions
type CSIGarbageCollector struct {
	apiReader    client.Reader
	fs           afero.Fs
	db           metadata.Cleaner
	mounter      mount.Interface
	time         *timeprovider.Provider
	isNotMounted mountChecker

	path metadata.PathResolver

	maxUnmountedVolumeAge time.Duration
}

// necessary for mocking, as the MounterMock will use the os package
type mountChecker func(mounter mount.Interface, file string) (bool, error)

var _ reconcile.Reconciler = (*CSIGarbageCollector)(nil)

const (
	safeRemovalThreshold = 5 * time.Minute
)

// NewCSIGarbageCollector returns a new CSIGarbageCollector
func NewCSIGarbageCollector(apiReader client.Reader, opts dtcsi.CSIOptions, db metadata.Cleaner) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		apiReader:             apiReader,
		fs:                    afero.NewOsFs(),
		db:                    db,
		path:                  metadata.PathResolver{RootDir: opts.RootDir},
		time:                  timeprovider.New(),
		mounter:               mount.New(""),
		isNotMounted:          mount.IsNotMountPoint,
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

		err := gc.runOSMountGarbageCollection(tenantConfig)
		if err != nil {
			continue
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

func (gc *CSIGarbageCollector) runOSMountGarbageCollection(tenantConfig metadata.TenantConfig) error {
	osMounts, err := gc.db.ListDeletedOSMounts()
	if err != nil {
		return err
	}

	for _, osm := range osMounts {
		if !gc.time.Now().Time.After(osm.DeletedAt.Time.Add(safeRemovalThreshold)) {
			log.Info("skipping recently removed os-mount", "location", osm.Location)

			continue
		}

		if osm.TenantConfig.UID == tenantConfig.UID {
			isNotMounted, err := gc.isNotMounted(gc.mounter, osm.Location)
			if err != nil {
				log.Info("failed to determine if OSMount is still mounted", "location", osm.Location, "tenantConfig", osm.TenantConfig.Name, "err", err.Error())

				continue
			}

			if !isNotMounted {
				log.Info("OSMount is still mounted", "location", osm.Location, "tenantConfig", osm.TenantConfig.Name)

				continue
			}

			dir, _ := afero.ReadDir(gc.fs, osm.Location)
			for _, d := range dir {
				gc.fs.RemoveAll(path.Join([]string{osm.Location, d.Name()}...))
				log.Info("removed outdate contents from OSMount folder", "location", osm.Location)
			}
		}
	}

	return nil
}
