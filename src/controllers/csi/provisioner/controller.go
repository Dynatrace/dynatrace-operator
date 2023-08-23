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

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csigc "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/gc"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/image"
	"github.com/Dynatrace/dynatrace-operator/src/installer/url"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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
	shortRequeueDuration   = 1 * time.Minute
	defaultRequeueDuration = 5 * time.Minute
	longRequeueDuration    = 30 * time.Minute
)

type urlInstallerBuilder func(afero.Fs, dtclient.Client, *url.Properties) installer.Installer
type imageInstallerBuilder func(afero.Fs, *image.Properties) installer.Installer

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client    client.Client
	apiReader client.Reader
	opts      dtcsi.CSIOptions
	fs        afero.Fs
	recorder  record.EventRecorder
	db        metadata.Access
	path      metadata.PathResolver
	gc        reconcile.Reconciler

	dynatraceClientBuilder dynatraceclient.Builder
	urlInstallerBuilder    urlInstallerBuilder
	imageInstallerBuilder  imageInstallerBuilder
}

// NewOneAgentProvisioner returns a new OneAgentProvisioner
func NewOneAgentProvisioner(mgr manager.Manager, opts dtcsi.CSIOptions, db metadata.Access) *OneAgentProvisioner {
	return &OneAgentProvisioner{
		client:                 mgr.GetClient(),
		apiReader:              mgr.GetAPIReader(),
		opts:                   opts,
		fs:                     afero.NewOsFs(),
		recorder:               mgr.GetEventRecorderFor("OneAgentProvisioner"),
		db:                     db,
		path:                   metadata.PathResolver{RootDir: opts.RootDir},
		gc:                     csigc.NewCSIGarbageCollector(mgr.GetAPIReader(), opts, db),
		dynatraceClientBuilder: dynatraceclient.NewBuilder(mgr.GetAPIReader()),
		urlInstallerBuilder:    url.NewUrlInstaller,
		imageInstallerBuilder:  image.NewImageInstaller,
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
			return reconcile.Result{}, provisioner.db.DeleteDynakube(ctx, request.Name)
		}
		return reconcile.Result{}, err
	}
	if !dk.NeedsCSIDriver() {
		log.Info("CSI driver provisioner not needed")
		return reconcile.Result{RequeueAfter: longRequeueDuration}, provisioner.db.DeleteDynakube(ctx, request.Name)
	}

	err = provisioner.setupFileSystem(dk)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !dk.NeedAppInjection() {
		log.Info("app injection not necessary, skip agent codemodule download", "dynakube", dk.Name)
		return reconcile.Result{RequeueAfter: longRequeueDuration}, nil
	}

	dynakubeMetadata, err := provisioner.setupDynakubeMetadata(ctx, dk) // needed for the CSI-resilience feature
	if err != nil {
		return reconcile.Result{}, err
	}

	if dk.CodeModulesImage() == "" && dk.CodeModulesVersion() == "" {
		log.Info("dynakube status is not yet ready, requeuing", "dynakube", dk.Name)
		return reconcile.Result{RequeueAfter: shortRequeueDuration}, err
	}

	err = provisioner.provisionCodeModules(ctx, dk, dynakubeMetadata)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = provisioner.collectGarbage(ctx, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func (provisioner *OneAgentProvisioner) setupFileSystem(dk *dynatracev1beta1.DynaKube) error {
	tenantUUID, err := dk.TenantUUIDFromApiUrl()
	if err != nil {
		return err
	}
	if err := provisioner.createCSIDirectories(tenantUUID); err != nil {
		log.Error(err, "error when creating csi directories", "path", provisioner.path.TenantDir(tenantUUID))
		return errors.WithStack(err)
	}
	log.Info("csi directories exist", "path", provisioner.path.TenantDir(tenantUUID))
	return nil
}

func (provisioner *OneAgentProvisioner) setupDynakubeMetadata(ctx context.Context, dk *dynatracev1beta1.DynaKube) (*metadata.Dynakube, error) {
	dynakubeMetadata, oldDynakubeMetadata, err := provisioner.handleMetadata(ctx, dk)
	if err != nil {
		return nil, err
	}

	// Create/update the dynakubeMetadata entry while `LatestVersion` is not necessarily set
	// so the host oneagent-storages can be mounted before the standalone agent binaries are ready to be mounted
	return dynakubeMetadata, provisioner.createOrUpdateDynakubeMetadata(ctx, oldDynakubeMetadata, dynakubeMetadata)
}

func (provisioner *OneAgentProvisioner) collectGarbage(ctx context.Context, request reconcile.Request) error {
	_, err := provisioner.gc.Reconcile(ctx, request)
	return err
}

func (provisioner *OneAgentProvisioner) provisionCodeModules(ctx context.Context, dk *dynatracev1beta1.DynaKube, dynakubeMetadata *metadata.Dynakube) error {
	oldDynakubeMetadata := *dynakubeMetadata
	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(provisioner, ctx, dk)
	if err != nil {
		return err
	}

	latestProcessModuleConfigCache, requeue, err := provisioner.updateAgentInstallation(ctx, dtc, dynakubeMetadata, dk)
	if requeue || err != nil {
		return err
	}

	// Set/Update the `LatestVersion` field in the database entry
	err = provisioner.createOrUpdateDynakubeMetadata(ctx, oldDynakubeMetadata, dynakubeMetadata)
	if err != nil {
		return err
	}

	err = provisioner.writeProcessModuleConfigCache(dynakubeMetadata.TenantUUID, latestProcessModuleConfigCache)
	if err != nil {
		return err
	}

	return nil
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

	tenantToken, err := provisioner.getAgentTenantToken(ctx, dk)
	if err != nil {
		return nil, false, err
	}

	latestProcessModuleConfig = latestProcessModuleConfig.
		AddHostGroup(dk.HostGroup()).
		AddConnectionInfo(dk.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")
	latestProcessModuleConfigCache = newProcessModuleConfigCache(latestProcessModuleConfig)

	if dk.NeedsOneAgentProxy() {
		proxy, err := dk.Proxy(ctx, provisioner.apiReader)
		if err != nil {
			return nil, false, err
		}
		latestProcessModuleConfig.AddProxy(proxy)
	}

	if dk.CodeModulesImage() != "" {
		updatedDigest, err := provisioner.installAgentImage(ctx, *dk, latestProcessModuleConfigCache)
		if err != nil {
			log.Info("error when updating agent from image", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return nil, true, nil
		} else if updatedDigest != "" {
			dynakubeMetadata.LatestVersion = ""
			dynakubeMetadata.ImageDigest = updatedDigest
		}
	} else {
		updateVersion, err := provisioner.installAgentZip(*dk, dtc, latestProcessModuleConfigCache)
		if err != nil {
			log.Info("error when updating agent from zip", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return nil, true, nil
		} else if updateVersion != "" {
			dynakubeMetadata.LatestVersion = updateVersion
			dynakubeMetadata.ImageDigest = ""
		}
	}
	return latestProcessModuleConfigCache, false, nil
}

func (provisioner *OneAgentProvisioner) getAgentTenantToken(ctx context.Context, dk *dynatracev1beta1.DynaKube) (string, error) {
	query := kubeobjects.NewSecretQuery(ctx, provisioner.client, provisioner.apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: dk.Namespace, Name: dk.OneagentTenantSecret()})
	if err != nil {
		return "", errors.Wrapf(err, "OneAgent tenant token secret %s/%s not found", dk.Namespace, dk.OneagentTenantSecret())
	}

	token, ok := secret.Data[connectioninfo.TenantTokenName]
	if !ok {
		return "", errors.Errorf("OneAgent tenant token not found in secret %s/%s", dk.Namespace, dk.OneagentTenantSecret())
	}

	return string(token), nil
}

func (provisioner *OneAgentProvisioner) handleMetadata(ctx context.Context, dk *dynatracev1beta1.DynaKube) (*metadata.Dynakube, metadata.Dynakube, error) {
	dynakubeMetadata, err := provisioner.db.GetDynakube(ctx, dk.Name)
	if err != nil {
		return nil, metadata.Dynakube{}, errors.WithStack(err)
	}

	// In case of a new dynakubeMetadata
	var oldDynakubeMetadata metadata.Dynakube
	if dynakubeMetadata != nil {
		oldDynakubeMetadata = *dynakubeMetadata
	}

	tenantUUID, err := dk.TenantUUIDFromApiUrl()
	if err != nil {
		return nil, metadata.Dynakube{}, err
	}

	dynakubeMetadata = metadata.NewDynakube(
		dk.Name,
		tenantUUID,
		oldDynakubeMetadata.LatestVersion,
		oldDynakubeMetadata.ImageDigest,
		dk.FeatureMaxFailedCsiMountAttempts())

	return dynakubeMetadata, oldDynakubeMetadata, nil
}

func (provisioner *OneAgentProvisioner) createOrUpdateDynakubeMetadata(ctx context.Context, oldDynakube metadata.Dynakube, dynakube *metadata.Dynakube) error {
	if oldDynakube != *dynakube {
		log.Info("dynakube has changed",
			"name", dynakube.Name,
			"tenantUUID", dynakube.TenantUUID,
			"version", dynakube.LatestVersion,
			"max mount attempts", dynakube.MaxFailedMountAttempts)
		if oldDynakube == (metadata.Dynakube{}) {
			log.Info("adding dynakube to db", "tenantUUID", dynakube.TenantUUID, "version", dynakube.LatestVersion)
			return provisioner.db.InsertDynakube(ctx, dynakube)
		} else {
			log.Info("updating dynakube in db",
				"old version", oldDynakube.LatestVersion, "new version", dynakube.LatestVersion,
				"old tenantUUID", oldDynakube.TenantUUID, "new tenantUUID", dynakube.TenantUUID)
			return provisioner.db.UpdateDynakube(ctx, dynakube)
		}
	}
	return nil
}

func buildDtc(provisioner *OneAgentProvisioner, ctx context.Context, dk *dynatracev1beta1.DynaKube) (dtclient.Client, error) {
	tokenReader := token.NewReader(provisioner.apiReader, dk)
	tokens, err := tokenReader.ReadTokens(ctx)

	if err != nil {
		return nil, err
	}

	dynatraceClient, err := provisioner.dynatraceClientBuilder.
		SetContext(ctx).
		SetDynakube(*dk).
		SetTokens(tokens).
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	return dynatraceClient, nil
}

func (provisioner *OneAgentProvisioner) getDynaKube(ctx context.Context, name types.NamespacedName) (*dynatracev1beta1.DynaKube, error) {
	var dk dynatracev1beta1.DynaKube
	err := provisioner.apiReader.Get(ctx, name, &dk)

	return &dk, err
}

func (provisioner *OneAgentProvisioner) createCSIDirectories(tenantUUID string) error {
	tenantDir := provisioner.path.TenantDir(tenantUUID)
	if err := provisioner.fs.MkdirAll(tenantDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", tenantDir, err)
	}

	agentBinaryDir := provisioner.path.AgentBinaryDir(tenantUUID)
	if err := provisioner.fs.MkdirAll(agentBinaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", agentBinaryDir, err)
	}

	return nil
}
