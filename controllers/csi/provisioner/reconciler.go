/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package csiprovisioner

import (
	"context"
	"fmt"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	failedInstallAgentVersionEvent = "FailedInstallAgentVersion"
	installAgentVersionEvent       = "InstallAgentVersion"
)

const (
	defaultRequeueDuration = 5 * time.Minute
	longRequeueDuration    = 30 * time.Minute
	shortRequeueDuration   = 15 * time.Second
)

var log = logger.NewDTLogger().WithName("provisioner")

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client       client.Client
	apiReader    client.Reader
	opts         dtcsi.CSIOptions
	dtcBuildFunc dynakube.DynatraceClientFunc
	fs           afero.Fs
	recorder     record.EventRecorder
	db           metadata.Access
	path         metadata.PathResolver
}

// NewReconciler returns a new OneAgentProvisioner
func NewReconciler(mgr manager.Manager, opts dtcsi.CSIOptions, db metadata.Access) *OneAgentProvisioner {
	return &OneAgentProvisioner{
		client:       mgr.GetClient(),
		apiReader:    mgr.GetAPIReader(),
		opts:         opts,
		dtcBuildFunc: dynakube.BuildDynatraceClient,
		fs:           afero.NewOsFs(),
		recorder:     mgr.GetEventRecorderFor("OneAgentProvisioner"),
		db:           db,
		path:         metadata.PathResolver{RootDir: opts.RootDir},
	}
}

func (r *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Complete(r)
}

var _ reconcile.Reconciler = &OneAgentProvisioner{}

func (r *OneAgentProvisioner) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := log.WithValues("namespace", request.Namespace, "name", request.Name)
	rlog.Info("Reconciling DynaKube")

	dk, err := r.getDynaKube(ctx, request.NamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, r.db.DeleteTenant(request.Name)
		}
		return reconcile.Result{}, err
	}
	if !dk.NeedsCSI() {
		rlog.Info("CSI driver not needed", "dynakube", dk.Name)
		return reconcile.Result{RequeueAfter: longRequeueDuration}, nil
	}

	if dk.ConnectionInfo().TenantUUID == "" {
		rlog.Info("DynaKube instance has not been reconciled yet and some values usually cached are missing, retrying in a few seconds")
		return reconcile.Result{RequeueAfter: shortRequeueDuration}, nil
	}

	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(r, ctx, dk)
	if err != nil {
		return reconcile.Result{}, err
	}

	tenant, err := r.db.GetTenant(dk.Name)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	// In case of a new tenant
	var oldTenant metadata.Tenant
	if tenant == nil {
		tenant = &metadata.Tenant{}
	} else {
		oldTenant = *tenant
	}
	rlog.Info("checking tenant", "dynakube", tenant.Dynakube,
		"uuid", tenant.TenantUUID, "version", tenant.LatestVersion)

	tenant.Dynakube = dk.Name
	tenant.TenantUUID = dk.ConnectionInfo().TenantUUID

	if err = r.createCSIDirectories(r.path.EnvDir(tenant.TenantUUID)); err != nil {
		rlog.Error(err, "error when creating csi directories", "path", r.path.EnvDir(tenant.TenantUUID))
		return reconcile.Result{}, errors.WithStack(err)
	}
	rlog.Info("csi directories exist", "path", r.path.EnvDir(tenant.TenantUUID))

	installAgentCfg := newInstallAgentConfig(rlog, dtc, r.path, r.fs, r.recorder, dk)
	if updatedVersion, err := installAgentCfg.updateAgent(tenant.LatestVersion, tenant.TenantUUID); err != nil {
		rlog.Info("error when updating agent", "error", err.Error())
		// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
		return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
	} else if updatedVersion != "" {
		tenant.LatestVersion = updatedVersion
	}

	err = r.createOrUpdateTenant(oldTenant, tenant)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func (r *OneAgentProvisioner) createOrUpdateTenant(oldTenant metadata.Tenant, tenant *metadata.Tenant) error {
	if hasTenantChanged(oldTenant, *tenant) {
		log.Info("tenant has changed",
			"dynakube", tenant.Dynakube, "uuid", tenant.TenantUUID, "version", tenant.LatestVersion)
		if oldTenant == (metadata.Tenant{}) {
			log.Info("Adding tenant",
				"dynakube", tenant.Dynakube, "uuid", tenant.TenantUUID, "version", tenant.LatestVersion)
			return r.db.InsertTenant(tenant)
		} else {
			log.Info("Updating tenant",
				"dynakube", tenant.Dynakube,
				"old version", oldTenant.LatestVersion, "new version", tenant.LatestVersion,
				"old tenant UUID", oldTenant.TenantUUID, "new tenant UUID", tenant.TenantUUID)
			return r.db.UpdateTenant(tenant)
		}
	}
	return nil
}

func hasTenantChanged(old, new metadata.Tenant) bool {
	return old != new
}

func buildDtc(r *OneAgentProvisioner, ctx context.Context, dk *dynatracev1beta1.DynaKube) (dtclient.Client, error) {
	dtp, err := dynakube.NewDynatraceClientProperties(ctx, r.apiReader, *dk)
	if err != nil {
		return nil, err
	}
	dtc, err := r.dtcBuildFunc(*dtp)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	return dtc, nil
}

func (r *OneAgentProvisioner) getDynaKube(ctx context.Context, name types.NamespacedName) (*dynatracev1beta1.DynaKube, error) {
	var dk dynatracev1beta1.DynaKube
	err := r.apiReader.Get(ctx, name, &dk)

	return &dk, err
}

func (r *OneAgentProvisioner) createCSIDirectories(envDir string) error {
	if err := r.fs.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", envDir, err)
	}

	return nil
}
