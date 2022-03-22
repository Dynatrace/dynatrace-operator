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
	"sync"
	"sync/atomic"
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

	eventRecorderName = "OneAgentProvisioner"
)

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client       client.Client
	apiReader    client.Reader
	opts         dtcsi.CSIOptions
	dtcBuildFunc dynakube.DynatraceClientFunc
	recorder     record.EventRecorder
	db           metadata.Access
	path         metadata.PathResolver

	mu                       *sync.Mutex
	currentParallelDownloads int64
	currentlyDownloading     map[string]bool
}

// NewOneAgentProvisioner returns a new OneAgentProvisioner
func NewOneAgentProvisioner(
	mgr manager.Manager,
	opts dtcsi.CSIOptions,
	db metadata.Access) *OneAgentProvisioner {
	return &OneAgentProvisioner{
		client:               mgr.GetClient(),
		apiReader:            mgr.GetAPIReader(),
		opts:                 opts,
		dtcBuildFunc:         dynakube.BuildDynatraceClient,
		recorder:             mgr.GetEventRecorderFor(eventRecorderName),
		db:                   db,
		path:                 metadata.PathResolver{RootDir: opts.RootDir},
		mu:                   &sync.Mutex{},
		currentlyDownloading: map[string]bool{},
	}
}

func (provisioner *OneAgentProvisioner) setCurrentlyDownloading(name string) {
	provisioner.mu.Lock()
	provisioner.currentlyDownloading[name] = true
	provisioner.mu.Unlock()
}

func (provisioner *OneAgentProvisioner) isCurrentlyDownloading(name string) bool {
	var isDownloading bool
	provisioner.mu.Lock()
	isDownloading = provisioner.currentlyDownloading[name]
	provisioner.mu.Unlock()
	return isDownloading
}

func (provisioner *OneAgentProvisioner) unsetCurrentlyDownloading(name string) {
	provisioner.mu.Lock()
	provisioner.currentlyDownloading[name] = false
	provisioner.mu.Unlock()
}

func (provisioner *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Complete(provisioner)
}

func (provisioner *OneAgentProvisioner) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	fs := afero.NewOsFs()

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

	if provisioner.parallelDownloadsEnabled() {
		if provisioner.parallelDownloadsLimitReached() {
			log.Info("max parallel downloads reached, requeued")
			return reconcile.Result{RequeueAfter: shortRequeueDuration}, nil
		}
		if provisioner.isCurrentlyDownloading(dk.ConnectionInfo().TenantUUID) {
			log.Info("still downloading")
			return reconcile.Result{RequeueAfter: shortRequeueDuration}, nil
		}

		atomic.AddInt64(&provisioner.currentParallelDownloads, 1)
		provisioner.setCurrentlyDownloading(dk.ConnectionInfo().TenantUUID)
		log.Info("staring parallel download")
		go provisioner.startParallelReconcile(ctx, request, fs, *dk)

		return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
	}
	return provisioner.reconcile(ctx, request, fs, *dk)
}

func (provisioner *OneAgentProvisioner) startParallelReconcile(ctx context.Context, request reconcile.Request, fs afero.Fs, dk dynatracev1beta1.DynaKube) {
	defer atomic.AddInt64(&provisioner.currentParallelDownloads, -1)
	defer provisioner.unsetCurrentlyDownloading(dk.ConnectionInfo().TenantUUID)
	_, err := provisioner.reconcile(ctx, request, fs, dk)
	if err != nil {
		log.Error(err, "Problem while provisioning oneagents in parallel")
	}
}

func (provisioner *OneAgentProvisioner) parallelDownloadsEnabled() bool {
	return provisioner.opts.MaxParallelDownloads > dtcsi.ParallelDownloadsLowerLimit
}
func (provisioner *OneAgentProvisioner) parallelDownloadsLimitReached() bool {
	return provisioner.currentParallelDownloads >= provisioner.opts.MaxParallelDownloads
}

func (provisioner *OneAgentProvisioner) reconcile(ctx context.Context, request reconcile.Request, fs afero.Fs, dynakube dynatracev1beta1.DynaKube) (reconcile.Result, error) {
	dynakubePointer := &dynakube
	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(provisioner, ctx, dynakubePointer)
	if err != nil {
		return reconcile.Result{}, err
	}

	dynakubeEntry, err := provisioner.db.GetDynakube(dynakubePointer.Name)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	// In case of a new dynakube
	var oldDynakube metadata.Dynakube
	if dynakubeEntry == nil {
		dynakubeEntry = &metadata.Dynakube{}
	} else {
		oldDynakube = *dynakubeEntry
	}
	log.Info("checking dynakube", "tenantUUID", dynakubeEntry.TenantUUID, "version", dynakubeEntry.LatestVersion)

	dynakubeEntry.Name = dynakubePointer.Name
	dynakubeEntry.TenantUUID = dynakubePointer.ConnectionInfo().TenantUUID

	// Create/update the dynakube entry while `LatestVersion` is not necessarily set
	// so the host oneagent-storages can be mounted before the standalone agent binaries are ready to be mounted
	err = provisioner.createOrUpdateDynakube(oldDynakube, dynakubeEntry)
	if err != nil {
		return reconcile.Result{}, err
	}
	oldDynakube = *dynakubeEntry

	if err = provisioner.createCSIDirectories(fs, provisioner.path.EnvDir(dynakubeEntry.TenantUUID)); err != nil {
		log.Error(err, "error when creating csi directories", "path", provisioner.path.EnvDir(dynakubeEntry.TenantUUID))
		return reconcile.Result{}, errors.WithStack(err)
	}
	log.Info("csi directories exist", "path", provisioner.path.EnvDir(dynakubeEntry.TenantUUID))

	latestProcessModuleConfig, storedHash, err := provisioner.getProcessModuleConfig(fs, dtc, dynakubeEntry.TenantUUID)
	if err != nil {
		log.Error(err, "error when getting the latest ruxitagentproc.conf")
		return reconcile.Result{}, err
	}
	latestProcessModuleConfigCache := newProcessModuleConfigCache(addHostGroup(dynakubePointer, latestProcessModuleConfig))

	installAgentCfg := newInstallAgentConfig(dtc, provisioner.path, fs, provisioner.recorder, dynakubePointer)
	if updatedVersion, err := installAgentCfg.updateAgent(dynakubeEntry.LatestVersion, dynakubeEntry.TenantUUID, storedHash, latestProcessModuleConfigCache); err != nil {
		log.Info("error when updating agent", "error", err.Error())
		// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
		return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
	} else if updatedVersion != "" {
		dynakubeEntry.LatestVersion = updatedVersion
	}

	// Set/Update the `LatestVersion` field in the database entry
	err = provisioner.createOrUpdateDynakube(oldDynakube, dynakubeEntry)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = provisioner.writeProcessModuleConfigCache(fs, dynakubeEntry.TenantUUID, latestProcessModuleConfigCache)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func (provisioner *OneAgentProvisioner) createOrUpdateDynakube(oldDynakube metadata.Dynakube, dynakube *metadata.Dynakube) error {
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

func (provisioner *OneAgentProvisioner) createCSIDirectories(fs afero.Fs, envDir string) error {
	if err := fs.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", envDir, err)
	}

	return nil
}
