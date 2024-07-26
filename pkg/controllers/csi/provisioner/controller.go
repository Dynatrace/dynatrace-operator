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

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csigc "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/gc"
	csiotel "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/internal/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	// "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	"github.com/spf13/afero"
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
	fs        afero.Afero
	recorder  record.EventRecorder
	gc        reconcile.Reconciler

	dynatraceClientBuilder dynatraceclient.Builder
	urlInstallerBuilder    urlInstallerBuilder
	imageInstallerBuilder  imageInstallerBuilder
	opts                   dtcsi.CSIOptions
	path                   metadata.PathResolver
}

// NewOneAgentProvisioner returns a new OneAgentProvisioner
func NewOneAgentProvisioner(mgr manager.Manager, opts dtcsi.CSIOptions) *OneAgentProvisioner {
	return &OneAgentProvisioner{
		client:                 mgr.GetClient(),
		apiReader:              mgr.GetAPIReader(),
		opts:                   opts,
		fs:                     afero.Afero{Fs: afero.NewOsFs()},
		recorder:               mgr.GetEventRecorderFor("OneAgentProvisioner"),
		path:                   metadata.PathResolver{RootDir: opts.RootDir},
		gc:                     csigc.NewCSIGarbageCollector(mgr.GetAPIReader(), opts),
		dynatraceClientBuilder: dynatraceclient.NewBuilder(mgr.GetAPIReader()),
		urlInstallerBuilder:    url.NewUrlInstaller,
		imageInstallerBuilder:  image.NewImageInstaller,
	}
}

func (provisioner *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynakube.DynaKube{}).
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

	if !dk.NeedAppInjection() {
		log.Info("app injection not necessary, skip agent codemodule download", "dynakube", dk.Name)

		return provisioner.collectGarbage(ctx, request)
	}

	if dk.CodeModulesImage() == "" && dk.CodeModulesVersion() == "" {
		log.Info("dynakube status is not yet ready, requeuing", "dynakube", dk.Name)

		return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
	}

	requeue, err := provisioner.provisionCodeModules(ctx, dk)
	if err != nil {
		if requeue {
			return reconcile.Result{RequeueAfter: dtcsi.ShortRequeueDuration}, err
		}

		return reconcile.Result{}, err
	}

	return provisioner.collectGarbage(ctx, request)
}

// needsReconcile checks if the DynaKube in the requests exists or needs any CSI functionality, if not then it runs the GC
func (provisioner *OneAgentProvisioner) needsReconcile(ctx context.Context, request reconcile.Request) (*dynakube.DynaKube, error) {
	dk, err := provisioner.getDynaKube(ctx, request.NamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("DynaKube was deleted, nothing to do")
		}

		return nil, nil //nolint: nilnil
	}

	if !dk.NeedsCSIDriver() {
		log.Info("CSI driver provisioner not needed")

		return nil, nil //nolint: nilnil
	}

	return dk, nil
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

func (provisioner *OneAgentProvisioner) provisionCodeModules(ctx context.Context, dk *dynakube.DynaKube) (requeue bool, err error) {
	// creates a dt client and checks tokens exist for the given dynakube
	dtc, err := buildDtc(provisioner, ctx, dk)
	if err != nil {
		return true, err
	}

	requeue, err = provisioner.updateAgentInstallation(ctx, dtc, dk)
	if err != nil {
		return requeue, err
	}

	return false, nil
}

func (provisioner *OneAgentProvisioner) updateAgentInstallation(
	ctx context.Context, dtc dtclient.Client,
	dk *dynakube.DynaKube,
) (
	requeue bool,
	err error,
) {
	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	// latestProcessModuleConfig, err := processmoduleconfigsecret.GetSecretData(ctx, provisioner.apiReader, dk.Name, dk.Namespace)
	// if err != nil {
	// 	span.RecordError(err)

	// 	return false, err
	// }

	if dk.CodeModulesImage() != "" {
		_, err := provisioner.installAgentImage(ctx, *dk)
		if err != nil {
			log.Info("error when updating agent from image", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return true, nil
		}
	} else {
		_, err := provisioner.installAgentZip(ctx, *dk, dtc)
		if err != nil {
			log.Info("error when updating agent from zip", "error", err.Error())
			// reporting error but not returning it to avoid immediate requeue and subsequently calling the API every few seconds
			return true, nil
		}
	}

	return false, nil
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
	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	var dk dynakube.DynaKube
	err := provisioner.apiReader.Get(ctx, name, &dk)
	span.RecordError(err)

	return &dk, err
}
