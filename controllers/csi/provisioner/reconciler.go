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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

	ruxitRevission, err := r.db.GetRuxitRevission(dynakube.TenantUUID)
	if err != nil {
		return reconcile.Result{}, err
	}
	if ruxitRevission == nil {
		ruxitRevission = metadata.NewRuxitRevission(dynakube.TenantUUID, 0)
	}

	ruxitConf, err := r.getRuxitProcConf(ruxitRevission, dtc)
	if err != nil {
		rlog.Error(err, "error when getting the latest ruxitagentproc.conf")
		return reconcile.Result{}, err
	}

	installAgentCfg := newInstallAgentConfig(rlog, dtc, r.path, r.fs, r.recorder, dk)
	if updatedVersion, err := installAgentCfg.updateAgent(dynakube.LatestVersion, dynakube.TenantUUID, ruxitRevission.LatestRevission, ruxitConf); err != nil {
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

	err = r.createOrUpdateRuxitRevision(dynakube.TenantUUID, ruxitRevission, ruxitConf)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func (r *OneAgentProvisioner) getRuxitProcConf(ruxitRevission *metadata.RuxitRevision, dtc dtclient.Client) (*dtclient.RuxitProcConf, error) {
	var latestRevission uint
	if ruxitRevission.LatestRevission != 0 {
		latestRevission = ruxitRevission.LatestRevission
	}

	ruxitConf, err := dtc.GetRuxitProcConf(latestRevission)
	if err != nil {
		return nil, err
	}
	if ruxitConf == nil {
		ruxitConf, err = r.readRuxitCache(ruxitRevission)
		if err != nil && os.IsNotExist(err) {
			ruxitConf, err = dtc.GetRuxitProcConf(0)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}
	return ruxitConf, r.writeRuxitCache(ruxitRevission, ruxitConf)
}

func (r *OneAgentProvisioner) readRuxitCache(ruxitRevission *metadata.RuxitRevision) (*dtclient.RuxitProcConf, error) {
	var ruxitConf dtclient.RuxitProcConf
	ruxitConfCache, err := r.fs.Open(r.path.AgentRuxitRevision(ruxitRevission.TenantUUID))
	if err != nil {
		return nil, err
	}
	jsonBytes, err := ioutil.ReadAll(ruxitConfCache)
	ruxitConfCache.Close()
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(jsonBytes, &ruxitConf); err != nil {
		return nil, err
	}
	return &ruxitConf, nil
}

func (r *OneAgentProvisioner) writeRuxitCache(ruxitRevission *metadata.RuxitRevision, ruxitConf *dtclient.RuxitProcConf) error {
	ruxitConfFile, err := r.fs.OpenFile(r.path.AgentRuxitRevision(ruxitRevission.TenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(ruxitConf)
	if err != nil {
		ruxitConfFile.Close()
		return err
	}
	_, err = ruxitConfFile.Write(jsonBytes)
	ruxitConfFile.Close()
	return err
}

func (r *OneAgentProvisioner) createOrUpdateRuxitRevision(tenantUUID string, ruxitRevision *metadata.RuxitRevision, ruxitConf *dtclient.RuxitProcConf) error {
	if ruxitRevision.LatestRevission == 0 && ruxitConf != nil {
		log.Info("inserting ruxit revission into db", "tenantUUID", tenantUUID, "revission", ruxitConf.Revision)
		return r.db.InsertRuxitRevission(metadata.NewRuxitRevission(tenantUUID, ruxitConf.Revision))
	} else if ruxitConf != nil && ruxitConf.Revision != ruxitRevision.LatestRevission {
		log.Info("updating ruxit revission in db", "tenantUUID", tenantUUID, "old-revission", ruxitRevision.LatestRevission, "new-revission", ruxitConf.Revision)
		ruxitRevision.LatestRevission = ruxitConf.Revision
		return r.db.UpdateRuxitRevission(ruxitRevision)
	}
	return nil
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
