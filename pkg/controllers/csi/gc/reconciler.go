package csigc

import (
	"context"
	"os"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CSIGarbageCollector removes unused and outdated agent versions
type CSIGarbageCollector struct {
	apiReader client.Reader
	fs        afero.Fs
	db        metadata.Access
	mounter   mount.Interface

	path metadata.PathResolver

	maxUnmountedVolumeAge time.Duration
}

var _ reconcile.Reconciler = (*CSIGarbageCollector)(nil)

// NewCSIGarbageCollector returns a new CSIGarbageCollector
func NewCSIGarbageCollector(apiReader client.Reader, opts dtcsi.CSIOptions, db metadata.Access) *CSIGarbageCollector {
	mounter := mount.New("")

	return &CSIGarbageCollector{
		apiReader:             apiReader,
		fs:                    afero.NewOsFs(),
		db:                    db,
		path:                  metadata.PathResolver{RootDir: opts.RootDir},
		mounter:               mounter,
		maxUnmountedVolumeAge: determineMaxUnmountedVolumeAge(os.Getenv(maxUnmountedCsiVolumeAgeEnv)),
	}
}

func (gc *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("running OneAgent garbage collection", "namespace", request.Namespace, "name", request.Name)

	defaultReconcileResult := reconcile.Result{}

	dynakube, err := getDynakubeFromRequest(ctx, gc.apiReader, request)
	if err != nil {
		return defaultReconcileResult, err
	}

	if dynakube == nil {
		return defaultReconcileResult, nil
	}

	if !dynakube.OneAgent().NeedAppInjection() {
		log.Info("app injection not enabled, skip garbage collection", "dynakube", dynakube.Name)

		return defaultReconcileResult, nil
	}

	tenantUUID, err := dynakube.TenantUUIDFromApiUrl()
	if err != nil {
		log.Info("failed to get tenantUUID of DynaKube, checking later")

		return defaultReconcileResult, err
	}

	log.Info("running log garbage collection")
	gc.runUnmountedVolumeGarbageCollection(tenantUUID)

	if err := ctx.Err(); err != nil {
		return defaultReconcileResult, err
	}

	log.Info("running binary garbage collection")

	if err := gc.runBinaryGarbageCollection(ctx, tenantUUID); err != nil {
		log.Info("failed to garbage collect the shared images")

		return defaultReconcileResult, err
	}

	return defaultReconcileResult, nil
}

func getDynakubeFromRequest(ctx context.Context, apiReader client.Reader, request reconcile.Request) (*dynakube.DynaKube, error) {
	var dk dynakube.DynaKube
	if err := apiReader.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("given DynaKube object not found")

			return nil, nil //nolint:nilnil
		}

		log.Info("failed to get DynaKube object")

		return nil, errors.WithStack(err)
	}

	return &dk, nil
}
