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
	"os"
	"path/filepath"
	"runtime"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logger.NewDTLogger().WithName("provisioner")

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client       client.Client
	opts         dtcsi.CSIOptions
	dtcBuildFunc dynakube.DynatraceClientFunc
	fs           afero.Fs
}

// NewReconciler returns a new OneAgentProvisioner
func NewReconciler(mgr manager.Manager, opts dtcsi.CSIOptions) *OneAgentProvisioner {
	return &OneAgentProvisioner{
		client:       mgr.GetClient(),
		opts:         opts,
		dtcBuildFunc: dynakube.BuildDynatraceClient,
		fs:           afero.NewOsFs(),
	}
}

func (r *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Complete(r)
}

var _ reconcile.Reconciler = &OneAgentProvisioner{}

func (r *OneAgentProvisioner) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := log.WithValues("namespace", request.Namespace, "name", request.Name)
	rlog.Info("Reconciling DynaKube")

	dk, err := getCodeModule(ctx, r.client, request.NamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	} else if dk == nil {
		rlog.Info("Code modules disabled")
		return reconcile.Result{RequeueAfter: 30 * time.Minute}, nil
	}

	dtc, err := buildDtc(r, ctx, dk)
	if err != nil {
		return reconcile.Result{}, err
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to fetch connection info: %w", err)
	}

	envDir := filepath.Join(r.opts.RootDir, dtcsi.DataPath, ci.TenantUUID)
	tenantFile := filepath.Join(r.opts.RootDir, dtcsi.DataPath, fmt.Sprintf("tenant-%s", dk.Name))

	if err = r.createCSIDirectories(envDir); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.updateTenantFile(ci.TenantUUID, tenantFile); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.updateAgent(dtc, envDir, rlog); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

func buildDtc(r *OneAgentProvisioner, ctx context.Context, dk *dynatracev1alpha1.DynaKube) (dtclient.Client, error) {
	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &tkns); err != nil {
		return nil, fmt.Errorf("failed to query tokens: %w", err)
	}

	dtc, err := r.dtcBuildFunc(r.client, dk, &tkns)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	return dtc, nil
}

func getCodeModule(ctx context.Context, clt client.Client, namespacedName types.NamespacedName) (*dynatracev1alpha1.DynaKube, error) {
	var dk dynatracev1alpha1.DynaKube
	if err := clt.Get(ctx, namespacedName, &dk); err != nil {
		return nil, err
	}

	if !dk.Spec.CodeModules.Enabled {
		return nil, nil
	}

	return &dk, nil
}

func (r *OneAgentProvisioner) updateAgent(dtc dtclient.Client, envDir string, logger logr.Logger) error {
	versionFile := filepath.Join(envDir, dtcsi.VersionDir)
	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return fmt.Errorf("failed to query OneAgent version: %w", err)
	}

	var oldVer string
	if b, err := afero.ReadFile(r.fs, versionFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to query installed OneAgent version: %w", err)
	} else {
		oldVer = string(b)
	}

	if ver != oldVer {
		if err = r.installAgentVersion(ver, envDir, dtc, logger); err != nil {
			return err
		}
	}

	return nil
}

func (r *OneAgentProvisioner) installAgentVersion(version string, envDir string, dtc dtclient.Client, logger logr.Logger) error {
	versionFile := filepath.Join(envDir, dtcsi.VersionDir)
	arch := dtclient.ArchX86
	if runtime.GOARCH == "arm64" {
		arch = dtclient.ArchARM
	}

	gcDir := filepath.Join(envDir, dtcsi.GarbageCollectionPath, version)
	if err := os.MkdirAll(gcDir, 0755); err != nil {
		logger.Error(err, "failed to create directory %s: %w", gcDir)
		return err
	}

	for _, flavor := range []string{dtclient.FlavorDefault, dtclient.FlavorMUSL} {
		targetDir := filepath.Join(envDir, "bin", version+"-"+flavor)

		if _, err := r.fs.Stat(targetDir); os.IsNotExist(err) {
			installAgentCfg := newInstallAgentConfig(logger, dtc, flavor, arch, targetDir)

			if err := installAgent(installAgentCfg); err != nil {
				if errDel := r.fs.RemoveAll(targetDir); errDel != nil {
					logger.Error(errDel, "failed to delete target directory", "path", targetDir)
				}

				return fmt.Errorf("failed to install agent: %w", err)
			}
		}
	}

	_ = afero.WriteFile(r.fs, versionFile, []byte(version), 0644)
	return nil
}

func (r *OneAgentProvisioner) updateTenantFile(tenantUUID string, tenantFile string) error {
	var oldDynaKubeTenant string
	if b, err := afero.ReadFile(r.fs, tenantFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to query assigned DynaKube tenant: %w", err)
	} else {
		oldDynaKubeTenant = string(b)
	}

	if oldDynaKubeTenant != tenantUUID {
		_ = afero.WriteFile(r.fs, tenantFile, []byte(tenantUUID), 0644)
	}

	return nil
}

func (r *OneAgentProvisioner) createCSIDirectories(envDir string) error {
	for _, dir := range []string{
		envDir,
		filepath.Join(envDir, dtcsi.LogDir),
		filepath.Join(envDir, dtcsi.DatastorageDir),
	} {
		if err := r.fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
