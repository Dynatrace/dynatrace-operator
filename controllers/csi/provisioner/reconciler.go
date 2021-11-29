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
			return reconcile.Result{}, r.db.DeleteDynakube(request.Name)
		}
		return reconcile.Result{}, err
	}
	if !dk.NeedsCSIDriver() {
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

	dynakube, err := r.db.GetDynakube(dk.Name)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	// In case of a new dynakube
	var oldDynakube metadata.Dynakube
	if dynakube == nil {
		dynakube = &metadata.Dynakube{}
	} else {
		oldDynakube = *dynakube
	}
	rlog.Info("checking dynakube", "name", dynakube.Name,
		"tenantUUID", dynakube.TenantUUID, "version", dynakube.LatestVersion)

	dynakube.Name = dk.Name
	dynakube.TenantUUID = dk.ConnectionInfo().TenantUUID

	if err = r.createCSIDirectories(r.path.EnvDir(dynakube.TenantUUID)); err != nil {
		rlog.Error(err, "error when creating csi directories", "path", r.path.EnvDir(dynakube.TenantUUID))
		return reconcile.Result{}, errors.WithStack(err)
	}
	rlog.Info("csi directories exist", "path", r.path.EnvDir(dynakube.TenantUUID))

	latestProcessModuleConfig, storedHash, err := r.getProcessModuleConfig(dtc, dynakube.TenantUUID)
	if err != nil {
		rlog.Error(err, "error when getting the latest ruxitagentproc.conf")
		return reconcile.Result{}, err
	}
	latestProcessModuleConfigCache := newProcessModuleConfigCache(addHostGroup(dk, latestProcessModuleConfig))

	installAgentCfg := newInstallAgentConfig(rlog, dtc, r.path, r.fs, r.recorder, dk)
	if updatedVersion, err := installAgentCfg.updateAgent(dynakube.LatestVersion, dynakube.TenantUUID, storedHash, latestProcessModuleConfigCache); err != nil {
		rlog.Info("error when updating agent", "error", err.Error())
		// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
		return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
	} else if updatedVersion != "" {
		dynakube.LatestVersion = updatedVersion
	}

	err = r.createOrUpdateDynakube(oldDynakube, dynakube)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.writeProcessModuleConfigCache(dynakube.TenantUUID, latestProcessModuleConfigCache)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func (r *OneAgentProvisioner) createOrUpdateDynakube(oldDynakube metadata.Dynakube, dynakube *metadata.Dynakube) error {
	if oldDynakube != *dynakube {
		log.Info("dynakube has changed",
			"name", dynakube.Name, "tenantUUID", dynakube.TenantUUID, "version", dynakube.LatestVersion)
		if oldDynakube == (metadata.Dynakube{}) {
			log.Info("adding dynakube to db",
				"name", dynakube.Name, "tenantUUID", dynakube.TenantUUID, "version", dynakube.LatestVersion)
			return r.db.InsertDynakube(dynakube)
		} else {
			log.Info("updating dynakube in db",
				"name", dynakube.Name,
				"old version", oldDynakube.LatestVersion, "new version", dynakube.LatestVersion,
				"old tenantUUID", oldDynakube.TenantUUID, "new tenantUUID", dynakube.TenantUUID)
			return r.db.UpdateDynakube(dynakube)
		}
	}
	return nil
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
