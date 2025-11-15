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
	"os"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/provisioner/cleanup"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/job"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/mount-utils"
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

type urlInstallerBuilder func(dtclient.Client, *url.Properties) installer.Installer
type imageInstallerBuilder func(context.Context, *image.Properties) (installer.Installer, error)
type jobInstallerBuilder func(context.Context, *job.Properties) installer.Installer

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	apiReader  client.Reader
	kubeClient client.Client

	dynatraceClientBuilder dynatraceclient.Builder
	urlInstallerBuilder    urlInstallerBuilder
	imageInstallerBuilder  imageInstallerBuilder
	jobInstallerBuilder    jobInstallerBuilder
	cleaner                *cleanup.Cleaner
	path                   metadata.PathResolver
}

// NewOneAgentProvisioner returns a new OneAgentProvisioner
func NewOneAgentProvisioner(mgr manager.Manager, opts dtcsi.CSIOptions) *OneAgentProvisioner {
	path := metadata.PathResolver{RootDir: opts.RootDir}

	return &OneAgentProvisioner{
		apiReader:              mgr.GetAPIReader(),
		kubeClient:             mgr.GetClient(),
		path:                   path,
		dynatraceClientBuilder: dynatraceclient.NewBuilder(mgr.GetAPIReader()),
		urlInstallerBuilder:    url.NewURLInstaller,
		imageInstallerBuilder:  image.NewImageInstaller,
		jobInstallerBuilder:    job.NewInstaller,
		cleaner:                cleanup.New(mgr.GetAPIReader(), path, mount.New("")),
	}
}

func (provisioner *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynakube.DynaKube{}).
		Owns(&batchv1.Job{}).
		Named("provisioner-controller").
		Complete(provisioner)
}

func (provisioner *OneAgentProvisioner) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling DynaKube", "namespace", request.Namespace, "dynakube", request.Name)

	var dk dynakube.DynaKube

	err := provisioner.apiReader.Get(ctx, request.NamespacedName, &dk)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = provisioner.cleaner.InstantRun(ctx)
			if err != nil {
				log.Error(err, "failed to run clean-up after dynakube deletion")
			}

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if !isProvisionerNeeded(&dk) {
		log.Info("CSI driver provisioner not needed")

		return reconcile.Result{RequeueAfter: longRequeueDuration}, provisioner.cleaner.Run(ctx)
	}

	err = provisioner.setupFileSystem(dk)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !dk.OneAgent().IsAppInjectionNeeded() {
		log.Info("app injection not necessary, skip agent codemodule download", "dynakube", dk.Name)

		_ = provisioner.cleaner.Run(ctx)

		return reconcile.Result{RequeueAfter: longRequeueDuration}, nil
	}

	if dk.OneAgent().GetCodeModulesImage() == "" && dk.OneAgent().GetCodeModulesVersion() == "" {
		log.Info("dynakube status is not yet ready, requeuing", "dynakube", dk.Name)

		return reconcile.Result{RequeueAfter: shortRequeueDuration}, nil
	}

	err = provisioner.installAgent(ctx, dk)
	if err != nil && errors.Is(err, errNotReady) {
		log.Info(err.Error(), "dynakube", dk.Name)

		return reconcile.Result{RequeueAfter: notReadyRequeueDuration}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	_ = provisioner.cleaner.Run(ctx)

	return reconcile.Result{RequeueAfter: defaultRequeueDuration}, nil
}

func isProvisionerNeeded(dk *dynakube.DynaKube) bool {
	return dk.OneAgent().IsAppInjectionNeeded() || dk.OneAgent().IsReadOnlyFSSupported()
}

func (provisioner *OneAgentProvisioner) setupFileSystem(dk dynakube.DynaKube) error {
	dynakubeDir := provisioner.path.DynaKubeDir(dk.GetName())
	if err := os.MkdirAll(dynakubeDir, 0755); err != nil {
		return errors.WithMessagef(err, "failed to create directory %s", dynakubeDir)
	}

	agentBinaryDir := provisioner.path.AgentSharedBinaryDirBase()
	if err := os.MkdirAll(agentBinaryDir, 0755); err != nil {
		return errors.WithMessagef(err, "failed to create directory %s", agentBinaryDir)
	}

	return nil
}

func buildDtc(provisioner *OneAgentProvisioner, ctx context.Context, dk dynakube.DynaKube) (dtclient.Client, error) {
	tokenReader := token.NewReader(provisioner.apiReader, &dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return nil, err
	}

	dynatraceClient, err := provisioner.dynatraceClientBuilder.
		SetDynakube(dk).
		SetTokens(tokens).
		Build(ctx)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create Dynatrace client")
	}

	return dynatraceClient, nil
}
