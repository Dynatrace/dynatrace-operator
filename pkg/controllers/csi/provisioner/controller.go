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

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csigc "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/gc"
	csiotel "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/internal/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
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

type urlInstallerBuilder func(afero.Fs, dtclient.Client, *url.Properties) installer.Installer
type imageInstallerBuilder func(context.Context, afero.Fs, *image.Properties) (installer.Installer, error)

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client    client.Client
	apiReader client.Reader
	fs        afero.Fs
	recorder  record.EventRecorder
	db        metadata.Access
	gc        reconcile.Reconciler

	dynatraceClientBuilder dynatraceclient.Builder
	urlInstallerBuilder    urlInstallerBuilder
	imageInstallerBuilder  imageInstallerBuilder
	opts                   dtcsi.CSIOptions
	path                   metadata.PathResolver
}

// NewOneAgentProvisioner returns a new OneAgentProvisioner
func NewOneAgentProvisioner(mgr manager.Manager, opts dtcsi.CSIOptions, db metadata.AccessCleaner) *OneAgentProvisioner {
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
		For(&dynatracev1beta2.DynaKube{}).
		Complete(provisioner)
}

func (provisioner *OneAgentProvisioner) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling DynaKube", "namespace", request.Namespace, "dynakube", request.Name)

	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	dk, err := provisioner.needsReconcile(ctx, request)
	if err != nil {
		span.RecordError(err)

		return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
	}

	if dk == nil {
		return provisioner.collectGarbage(ctx, request)
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

		return provisioner.collectGarbage(ctx, request)
	}

	if dk.CodeModulesImage() == "" && dk.CodeModulesVersion() == "" {
		log.Info("dynakube status is not yet ready, requeuing", "dynakube", dk.Name)

		return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
	}

	requeue, err := provisioner.provisionCodeModules(ctx, dk, tenantConfig)
	if err != nil {
		if requeue {
			return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
		}

		return reconcile.Result{}, err
	}

	return provisioner.collectGarbage(ctx, request)
}

// needsReconcile checks if the DynaKube in the requests exists or needs any CSI functionality, if not then it runs the GC
func (provisioner *OneAgentProvisioner) needsReconcile(ctx context.Context, request reconcile.Request) (*dynatracev1beta2.DynaKube, error) {
	dk, err := provisioner.getDynaKube(ctx, request.NamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("DynaKube was deleted, running cleanup")

			err := provisioner.db.DeleteTenantConfig(&metadata.TenantConfig{Name: request.Name}, true)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil //nolint: nilnil
	}

	if !dk.NeedsCSIDriver() {
		log.Info("CSI driver provisioner not needed")

		err = provisioner.db.DeleteTenantConfig(&metadata.TenantConfig{Name: dk.Name}, true)
		if err != nil {
			return nil, err
		}

		return nil, nil //nolint: nilnil
	}

	return dk, nil
}

func (provisioner *OneAgentProvisioner) setupFileSystem(dk *dynatracev1beta2.DynaKube) error {
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

func (provisioner *OneAgentProvisioner) setupTenantConfig(dk *dynatracev1beta2.DynaKube) (*metadata.TenantConfig, error) {
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

func (provisioner *OneAgentProvisioner) collectGarbage(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	result, err := provisioner.gc.Reconcile(ctx, request)

	if err != nil {
		span.RecordError(err)

		return result, err
	}

	return result, nil
}

func (provisioner *OneAgentProvisioner) provisionCodeModules(ctx context.Context, dk *dynatracev1beta2.DynaKube, tenantConfig *metadata.TenantConfig) (requeue bool, err error) {
	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(provisioner, ctx, dk)
	if err != nil {
		return true, err
	}

	requeue, err = provisioner.updateAgentInstallation(ctx, dtc, tenantConfig, dk)
	if err != nil {
		return requeue, err
	}

	// Set/Update the `LatestVersion` field in the database entry
	err = provisioner.db.UpdateTenantConfig(tenantConfig)
	if err != nil {
		return true, err
	}

	return false, nil
}

func (provisioner *OneAgentProvisioner) updateAgentInstallation(
	ctx context.Context, dtc dtclient.Client,
	tenantConfig *metadata.TenantConfig,
	dk *dynatracev1beta2.DynaKube,
) (
	requeue bool,
	err error,
) {
	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	latestProcessModuleConfig, err := processmoduleconfigsecret.GetSecretData(ctx, provisioner.apiReader, dk.Name, dk.Namespace)
	if err != nil {
		span.RecordError(err)

		return false, err
	}

	if dk.CodeModulesImage() != "" {
		updatedImageURI, err := provisioner.installAgentImage(ctx, *dk, latestProcessModuleConfig)
		if err != nil {
			log.Info("error when updating agent from image", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return true, nil
		}

		tenantConfig.DownloadedCodeModuleVersion = updatedImageURI
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

func (provisioner *OneAgentProvisioner) handleMetadata(dk *dynatracev1beta2.DynaKube) (*metadata.TenantConfig, error) {
	tenantUUID, err := dk.TenantUUIDFromApiUrl() // TODO update to use the tenant uuid from the DynaKube status
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

func buildDtc(provisioner *OneAgentProvisioner, ctx context.Context, dk *dynatracev1beta2.DynaKube) (dtclient.Client, error) {
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

func (provisioner *OneAgentProvisioner) getDynaKube(ctx context.Context, name types.NamespacedName) (*dynatracev1beta2.DynaKube, error) {
	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	var dk dynatracev1beta2.DynaKube
	err := provisioner.apiReader.Get(ctx, name, &dk)
	span.RecordError(err)

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
