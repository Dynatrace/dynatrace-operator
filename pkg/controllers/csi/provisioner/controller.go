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

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csigc "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/gc"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"gorm.io/gorm"
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
type imageInstallerBuilder func(afero.Fs, *image.Properties) (installer.Installer, error)

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client    client.Client
	apiReader client.Reader
	fs        afero.Fs
	recorder  record.EventRecorder
	db        metadata.GormAccess
	gc        reconcile.Reconciler

	dynatraceClientBuilder dynatraceclient.Builder
	urlInstallerBuilder    urlInstallerBuilder
	imageInstallerBuilder  imageInstallerBuilder
	registryClientBuilder  registry.ClientBuilder
	opts                   dtcsi.CSIOptions
	path                   metadata.PathResolver
}

// NewOneAgentProvisioner returns a new OneAgentProvisioner
func NewOneAgentProvisioner(mgr manager.Manager, opts dtcsi.CSIOptions, db metadata.GormAccess) *OneAgentProvisioner {
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
		registryClientBuilder:  registry.NewClient,
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
			return reconcile.Result{}, provisioner.db.DeleteTenantConfig(&metadata.TenantConfig{Name: request.Name}, false)
		}

		return reconcile.Result{}, err
	}

	if !dk.NeedsCSIDriver() {
		log.Info("CSI driver provisioner not needed")

		err = provisioner.db.DeleteTenantConfig(&metadata.TenantConfig{Name: dk.Name}, true)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{RequeueAfter: longRequeueDuration}, err
	}

	err = provisioner.setupFileSystem(dk)
	if err != nil {
		return reconcile.Result{}, err
	}

	tenantConfig, err := provisioner.setupTenantConfig(dk) // needed for the CSI-resilience feature
	if err != nil {
		return reconcile.Result{}, err
	}

	if !dk.NeedAppInjection() {
		log.Info("app injection not necessary, skip agent codemodule download", "dynakube", dk.Name)

		return reconcile.Result{RequeueAfter: longRequeueDuration}, nil
	}

	if dk.CodeModulesImage() == "" && dk.CodeModulesVersion() == "" {
		log.Info("dynakube status is not yet ready, requeuing", "dynakube", dk.Name)

		return reconcile.Result{RequeueAfter: shortRequeueDuration}, err
	}

	err = provisioner.provisionCodeModules(ctx, dk, tenantConfig)
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

func (provisioner *OneAgentProvisioner) setupTenantConfig(dk *dynatracev1beta1.DynaKube) (*metadata.TenantConfig, error) {
	metadataTenantConfig, err := provisioner.handleMetadata(dk)
	if err != nil {
		return nil, err
	}

	// Create/update the Dynakube's metadata TenantConfig entry while `LatestVersion` is not necessarily set
	// so the host oneagent-storages can be mounted before the standalone agent binaries are ready to be mounted
	tenantConfig, err := provisioner.db.ReadTenantConfig(metadata.TenantConfig{Name: metadataTenantConfig.Name})
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err = provisioner.db.CreateTenantConfig(metadataTenantConfig)
		if err != nil {
			return nil, err
		}

		return metadataTenantConfig, nil
	} else if err != nil {
		return nil, err
	}

	metadataTenantConfig.UID = tenantConfig.UID
	err = provisioner.db.UpdateTenantConfig(metadataTenantConfig)

	if err != nil {
		return nil, err
	}

	return metadataTenantConfig, nil
}

func (provisioner *OneAgentProvisioner) collectGarbage(ctx context.Context, request reconcile.Request) error {
	_, err := provisioner.gc.Reconcile(ctx, request)

	return err
}

func (provisioner *OneAgentProvisioner) provisionCodeModules(ctx context.Context, dk *dynatracev1beta1.DynaKube, tenantConfig *metadata.TenantConfig) error {
	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(provisioner, ctx, dk)
	if err != nil {
		return err
	}

	requeue, err := provisioner.updateAgentInstallation(ctx, dtc, tenantConfig, dk)
	if requeue || err != nil {
		return err
	}

	// Set/Update the `LatestVersion` field in the database entry
	err = provisioner.db.UpdateTenantConfig(tenantConfig)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *OneAgentProvisioner) updateAgentInstallation(
	ctx context.Context, dtc dtclient.Client,
	tenantConfig *metadata.TenantConfig,
	dk *dynatracev1beta1.DynaKube,
) (
	requeue bool,
	err error,
) {
	latestProcessModuleConfig, err := processmoduleconfigsecret.GetSecretData(ctx, provisioner.apiReader, dk.Name, dk.Namespace)
	if err != nil {
		return false, err
	}

	if dk.CodeModulesImage() != "" {
		updatedDigest, err := provisioner.installAgentImage(ctx, *dk, latestProcessModuleConfig)
		if err != nil {
			log.Info("error when updating agent from image", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return true, nil
		}

		tenantConfig.DownloadedCodeModuleVersion = updatedDigest
	} else {
		updateVersion, err := provisioner.installAgentZip(ctx, *dk, dtc, latestProcessModuleConfig)
		if err != nil {
			log.Info("error when updating agent from zip", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return true, nil
		}

		if updateVersion != "" {
			tenantConfig.DownloadedCodeModuleVersion = updateVersion
		}
	}

	return false, nil
}

func (provisioner *OneAgentProvisioner) handleMetadata(dk *dynatracev1beta1.DynaKube) (*metadata.TenantConfig, error) {
	tenantUUID, err := dk.TenantUUIDFromApiUrl()
	if err != nil {
		return nil, err
	}

	newTenantConfig := &metadata.TenantConfig{
		UID:                         string(dk.UID),
		Name:                        dk.Name,
		TenantUUID:                  tenantUUID,
		DownloadedCodeModuleVersion: dk.CodeModulesVersion(),
		MaxFailedMountAttempts:      int64(dk.FeatureMaxFailedCsiMountAttempts()),
		ConfigDirPath:               provisioner.path.AgentConfigDir(tenantUUID, dk.Name),
	}

	return newTenantConfig, nil
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
