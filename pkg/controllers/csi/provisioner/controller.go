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

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
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
type imageInstallerBuilder func(context.Context, afero.Fs, *image.Properties) (installer.Installer, error)

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client    client.Client
	apiReader client.Reader
	fs        afero.Fs
	recorder  record.EventRecorder
	db        metadata.Access

	dynatraceClientBuilder dynatraceclient.Builder
	urlInstallerBuilder    urlInstallerBuilder
	imageInstallerBuilder  imageInstallerBuilder
	opts                   dtcsi.CSIOptions
	path                   metadata.PathResolver
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
		dynatraceClientBuilder: dynatraceclient.NewBuilder(mgr.GetAPIReader()),
		urlInstallerBuilder:    url.NewUrlInstaller,
		imageInstallerBuilder:  image.NewImageInstaller,
	}
}

func (provisioner *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynakube.DynaKube{}).
		Named("provisioner-controller").
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

	dynakubeMetadata, err := provisioner.setupDynakubeMetadata(ctx, dk) // needed for the CSI-resilience feature
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

	err = provisioner.provisionCodeModules(ctx, dk, dynakubeMetadata)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func (provisioner *OneAgentProvisioner) setupFileSystem(dk *dynakube.DynaKube) error {
	if err := provisioner.createCSIDirectories(dk.GetName()); err != nil {
		log.Error(err, "error when creating csi directories", "path", provisioner.path.DynaKubeDir(dk.GetName()))

		return errors.WithStack(err)
	}

	log.Info("csi directories exist", "path", provisioner.path.DynaKubeDir(dk.GetName()))

	return nil
}

func (provisioner *OneAgentProvisioner) setupDynakubeMetadata(ctx context.Context, dk *dynakube.DynaKube) (*metadata.Dynakube, error) {
	dynakubeMetadata, oldDynakubeMetadata, err := provisioner.handleMetadata(ctx, dk)
	if err != nil {
		return nil, err
	}

	// Create/update the dynakubeMetadata entry while `LatestVersion` is not necessarily set
	// so the host oneagent-storages can be mounted before the standalone agent binaries are ready to be mounted
	return dynakubeMetadata, provisioner.createOrUpdateDynakubeMetadata(ctx, oldDynakubeMetadata, dynakubeMetadata)
}

func (provisioner *OneAgentProvisioner) provisionCodeModules(ctx context.Context, dk *dynakube.DynaKube, dynakubeMetadata *metadata.Dynakube) error {
	oldDynakubeMetadata := *dynakubeMetadata
	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(provisioner, ctx, dk)
	if err != nil {
		return err
	}

	requeue, err := provisioner.updateAgentInstallation(ctx, dtc, dynakubeMetadata, dk)
	if requeue || err != nil {
		return err
	}

	// Set/Update the `LatestVersion` field in the database entry
	err = provisioner.createOrUpdateDynakubeMetadata(ctx, oldDynakubeMetadata, dynakubeMetadata)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *OneAgentProvisioner) updateAgentInstallation(
	ctx context.Context, dtc dtclient.Client,
	dynakubeMetadata *metadata.Dynakube,
	dk *dynakube.DynaKube,
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
		} else if updatedDigest != "" {
			dynakubeMetadata.LatestVersion = ""
			dynakubeMetadata.ImageDigest = updatedDigest
		}
	} else {
		updateVersion, err := provisioner.installAgentZip(ctx, *dk, dtc, latestProcessModuleConfig)
		if err != nil {
			log.Info("error when updating agent from zip", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return true, nil
		} else if updateVersion != "" {
			dynakubeMetadata.LatestVersion = updateVersion
			dynakubeMetadata.ImageDigest = ""
		}
	}

	return false, nil
}

func (provisioner *OneAgentProvisioner) handleMetadata(ctx context.Context, dk *dynakube.DynaKube) (*metadata.Dynakube, metadata.Dynakube, error) {
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

func buildDtc(provisioner *OneAgentProvisioner, ctx context.Context, dk *dynakube.DynaKube) (dtclient.Client, error) {
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

func (provisioner *OneAgentProvisioner) getDynaKube(ctx context.Context, name types.NamespacedName) (*dynakube.DynaKube, error) {
	var dk dynakube.DynaKube
	err := provisioner.apiReader.Get(ctx, name, &dk)

	return &dk, err
}

func (provisioner *OneAgentProvisioner) createCSIDirectories(dynakubeName string) error {
	dynakubeDir := provisioner.path.DynaKubeDir(dynakubeName)
	if err := provisioner.fs.MkdirAll(dynakubeDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dynakubeDir, err)
	}

	agentBinaryDir := provisioner.path.AgentBinaryDir(dynakubeName)
	if err := provisioner.fs.MkdirAll(agentBinaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", agentBinaryDir, err)
	}

	return nil
}
