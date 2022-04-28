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

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
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
	defaultRequeueDuration = 5 * time.Minute
	longRequeueDuration    = 30 * time.Minute
	shortRequeueDuration   = 15 * time.Second
)

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

// NewOneAgentProvisioner returns a new OneAgentProvisioner
func NewOneAgentProvisioner(mgr manager.Manager, opts dtcsi.CSIOptions, db metadata.Access) *OneAgentProvisioner {
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

func (provisioner *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Complete(provisioner)
}

func (provisioner *OneAgentProvisioner) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling DynaKube", "namespace", request.Namespace, "dynakube", request.Name)

	dk, err := provisioner.getDynaKube(ctx, request.NamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, provisioner.db.DeleteDynakube(request.Name)
		}
		return reconcile.Result{}, err
	}
	if !dk.NeedsCSIDriver() {
		log.Info("CSI driver not needed")
		return reconcile.Result{RequeueAfter: longRequeueDuration}, nil
	}

	if dk.ConnectionInfo().TenantUUID == "" {
		log.Info("DynaKube instance has not been reconciled yet and some values usually cached are missing, retrying in a few seconds")
		return reconcile.Result{RequeueAfter: shortRequeueDuration}, nil
	}

	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(provisioner, ctx, dk)
	if err != nil {
		return reconcile.Result{}, err
	}

	dynakubeMetadata, oldDynakubeMetadata, err := provisioner.handleMetadata(dk)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("checking dynakube", "tenantUUID", dynakubeMetadata.TenantUUID, "version", dynakubeMetadata.LatestVersion)

	// Create/update the dynakubeMetadata entry while `LatestVersion` is not necessarily set
	// so the host oneagent-storages can be mounted before the standalone agent binaries are ready to be mounted
	err = provisioner.createOrUpdateDynakubeMetadata(oldDynakubeMetadata, dynakubeMetadata)
	if err != nil {
		return reconcile.Result{}, err
	}
	oldDynakubeMetadata = *dynakubeMetadata

	if err = provisioner.createCSIDirectories(provisioner.path.EnvDir(dynakubeMetadata.TenantUUID)); err != nil {
		log.Error(err, "error when creating csi directories", "path", provisioner.path.EnvDir(dynakubeMetadata.TenantUUID))
		return reconcile.Result{}, errors.WithStack(err)
	}
	log.Info("csi directories exist", "path", provisioner.path.EnvDir(dynakubeMetadata.TenantUUID))

	latestProcessModuleConfigCache, requeue, err := provisioner.updateAgentInstallation(ctx, dtc, dynakubeMetadata, dk)
	if requeue {
		return reconcile.Result{RequeueAfter: defaultRequeueDuration}, err
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	// Set/Update the `LatestVersion` field in the database entry
	err = provisioner.createOrUpdateDynakubeMetadata(oldDynakubeMetadata, dynakubeMetadata)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = provisioner.writeProcessModuleConfigCache(dynakubeMetadata.TenantUUID, latestProcessModuleConfigCache)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func (provisioner *OneAgentProvisioner) updateAgentInstallation(ctx context.Context, dtc dtclient.Client, dynakubeMetadata *metadata.Dynakube, dk *dynatracev1beta1.DynaKube) (
	latestProcessModuleConfigCache *processModuleConfigCache,
	requeue bool,
	err error,
) {
	latestProcessModuleConfig, _, err := provisioner.getProcessModuleConfig(dtc, dynakubeMetadata.TenantUUID)
	if err != nil {
		log.Error(err, "error when getting the latest ruxitagentproc.conf")
		return nil, false, err
	}
	latestProcessModuleConfig = latestProcessModuleConfig.AddHostGroup(dk.HostGroup())

	if dk.CodeModulesImage() != "" {
		connectionInfo, err := dtc.GetConnectionInfo()
		if err != nil {
			log.Error(err, "error when getting OneAgent connectionInfo")
			return nil, false, err
		}
		latestProcessModuleConfig = latestProcessModuleConfig.AddConnectionInfo(connectionInfo)
	}

	latestProcessModuleConfigCache = newProcessModuleConfigCache(latestProcessModuleConfig)

	agentUpdater, err := newAgentUpdater(ctx, provisioner.fs, provisioner.apiReader, dtc, provisioner.path, provisioner.recorder, dk)
	if err != nil {
		log.Info("error when setting up the agent updater", "error", err.Error())
		return nil, false, err
	}
	if updatedVersion, err := agentUpdater.updateAgent(dynakubeMetadata.LatestVersion, dynakubeMetadata.TenantUUID, latestProcessModuleConfigCache); err != nil {
		log.Info("error when updating agent", "error", err.Error())
		// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
		return nil, true, nil
	} else if updatedVersion != "" {
		dynakubeMetadata.LatestVersion = updatedVersion
	}
	return latestProcessModuleConfigCache, false, nil
}

func (provisioner *OneAgentProvisioner) handleMetadata(dk *dynatracev1beta1.DynaKube) (*metadata.Dynakube, metadata.Dynakube, error) {
	dynakubeMetadata, err := provisioner.db.GetDynakube(dk.Name)
	if err != nil {
		return nil, metadata.Dynakube{}, errors.WithStack(err)
	}

	// In case of a new dynakubeMetadata
	var oldDynakubeMetadata metadata.Dynakube
	if dynakubeMetadata == nil {
		dynakubeMetadata = &metadata.Dynakube{}
	} else {
		oldDynakubeMetadata = *dynakubeMetadata
	}

	dynakubeMetadata.Name = dk.Name
	dynakubeMetadata.TenantUUID = dk.ConnectionInfo().TenantUUID

	return dynakubeMetadata, oldDynakubeMetadata, nil
}

func (provisioner *OneAgentProvisioner) createOrUpdateDynakubeMetadata(oldDynakube metadata.Dynakube, dynakube *metadata.Dynakube) error {
	if oldDynakube != *dynakube {
		log.Info("dynakube has changed",
			"name", dynakube.Name, "tenantUUID", dynakube.TenantUUID, "version", dynakube.LatestVersion)
		if oldDynakube == (metadata.Dynakube{}) {
			log.Info("adding dynakube to db", "tenantUUID", dynakube.TenantUUID, "version", dynakube.LatestVersion)
			return provisioner.db.InsertDynakube(dynakube)
		} else {
			log.Info("updating dynakube in db",
				"old version", oldDynakube.LatestVersion, "new version", dynakube.LatestVersion,
				"old tenantUUID", oldDynakube.TenantUUID, "new tenantUUID", dynakube.TenantUUID)
			return provisioner.db.UpdateDynakube(dynakube)
		}
	}
	return nil
}

func buildDtc(provisioner *OneAgentProvisioner, ctx context.Context, dk *dynatracev1beta1.DynaKube) (dtclient.Client, error) {
	dtp, err := dynakube.NewDynatraceClientProperties(ctx, provisioner.apiReader, *dk)
	if err != nil {
		return nil, err
	}
	dtc, err := provisioner.dtcBuildFunc(*dtp)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	return dtc, nil
}

func (provisioner *OneAgentProvisioner) getDynaKube(ctx context.Context, name types.NamespacedName) (*dynatracev1beta1.DynaKube, error) {
	var dk dynatracev1beta1.DynaKube
	err := provisioner.apiReader.Get(ctx, name, &dk)

	return &dk, err
}

func (provisioner *OneAgentProvisioner) createCSIDirectories(envDir string) error {
	if err := provisioner.fs.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", envDir, err)
	}

	return nil
}
