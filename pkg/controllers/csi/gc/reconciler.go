package csigc

import (
	"context"
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
	mounter      mount.Interface
	time         *timeprovider.Provider
	isNotMounted mountChecker

	path metadata.PathResolver
}

// necessary for mocking, as the MounterMock will use the os package
type mountChecker func(mounter mount.Interface, file string) (bool, error)

var _ reconcile.Reconciler = (*CSIGarbageCollector)(nil)

const (
	safeRemovalThreshold = 5 * time.Minute
)

// NewCSIGarbageCollector returns a new CSIGarbageCollector
func NewCSIGarbageCollector(apiReader client.Reader, opts dtcsi.CSIOptions) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		apiReader:             apiReader,
		fs:                    afero.NewOsFs(),
		path:                  metadata.PathResolver{RootDir: opts.RootDir},
		time:                  timeprovider.New(),
		mounter:               mount.New(""),
		isNotMounted:          mount.IsNotMountPoint,
	}
}

func (gc *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// TODO: Reimplement from scratch
	return reconcile.Result{}, nil
}
