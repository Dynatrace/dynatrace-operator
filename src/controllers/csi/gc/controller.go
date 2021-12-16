package csigc

import (
	"context"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/spf13/afero"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CSIGarbageCollector removes unused and outdated agent versions
type CSIGarbageCollector struct {
	apiReader    client.Reader
	opts         dtcsi.CSIOptions
	dtcBuildFunc dynakube.DynatraceClientFunc
	fs           afero.Fs
	db           metadata.Access
	path         metadata.PathResolver
}

// NewController returns a new CSIGarbageCollector
func NewController(apiReader client.Reader, opts dtcsi.CSIOptions, db metadata.Access) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		apiReader:    apiReader,
		opts:         opts,
		dtcBuildFunc: dynakube.BuildDynatraceClient,
		fs:           afero.NewOsFs(),
		db:           db,
		path:         metadata.PathResolver{RootDir: opts.RootDir},
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

	var dk dynatracev1beta1.DynaKube
	if err := gc.apiReader.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("given DynaKube object not found")
			return reconcileResult, nil
		}

		log.Error(err, "failed to get DynaKube object")
		return reconcileResult, nil
	}

	dtp, err := dynakube.NewDynatraceClientProperties(ctx, gc.apiReader, dk)
	if err != nil {
		log.Error(err, err.Error())
		return reconcileResult, nil
	}

	dtc, err := gc.dtcBuildFunc(*dtp)
	if err != nil {
		log.Error(err, "failed to create Dynatrace client")
		return reconcileResult, nil
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		log.Info("failed to fetch connection info")
		return reconcileResult, nil
	}

	latestAgentVersion, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		log.Info("failed to query OneAgent version")
		return reconcileResult, nil
	}

	log.Info("running binary garbage collection")
	gc.runBinaryGarbageCollection(ci.TenantUUID, latestAgentVersion)

	log.Info("running log garbage collection")
	gc.runLogGarbageCollection(ci.TenantUUID)

	return reconcileResult, nil
}
