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
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logger.NewDTLogger().WithName("provisioner")

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	client        client.Client
	opts          dtcsi.CSIOptions
	dtcBuildFunc  dynakube.DynatraceClientFunc
	mkDirAllFunc  func(path string, perm fs.FileMode) error
	writeFileFunc func(filename string, data []byte, perm fs.FileMode) error
	readFileFunc  func(filename string) ([]byte, error)
}

// NewReconciler returns a new OneAgentProvisioner
func NewReconciler(mgr manager.Manager, opts dtcsi.CSIOptions) *OneAgentProvisioner {
	return &OneAgentProvisioner{
		client:        mgr.GetClient(),
		opts:          opts,
		dtcBuildFunc:  dynakube.BuildDynatraceClient,
		mkDirAllFunc:  os.MkdirAll,
		writeFileFunc: ioutil.WriteFile,
		readFileFunc:  ioutil.ReadFile,
	}
}

func (r *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Complete(r)
}

var _ reconcile.Reconciler = &OneAgentProvisioner{}

func (r *OneAgentProvisioner) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rlog := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rlog.Info("Reconciling DynaKube")

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !dk.Spec.CodeModules.Enabled {
		rlog.Info("Code modules disabled")
		return reconcile.Result{RequeueAfter: 30 * time.Minute}, nil
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &tkns); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query tokens: %w", err)
	}

	dtc, err := r.dtcBuildFunc(r.client, &dk, &tkns)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to fetch connection info: %w", err)
	}

	envDir := filepath.Join(r.opts.DataDir, ci.TenantUUID)
	verFile := filepath.Join(envDir, "version")
	tenantFile := filepath.Join(r.opts.DataDir, fmt.Sprintf("tenant-%s", dk.Name))

	for _, dir := range []string{
		envDir,
		filepath.Join(envDir, "log"),
		filepath.Join(envDir, "datastorage"),
	} {
		if err := r.mkDirAllFunc(dir, 0755); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	var oldDynaKubeTenant string
	if b, err := r.readFileFunc(tenantFile); err != nil && !os.IsNotExist(err) {
		return reconcile.Result{}, fmt.Errorf("failed to query assigned DynaKube tenant: %w", err)
	} else {
		oldDynaKubeTenant = string(b)
	}

	if oldDynaKubeTenant != ci.TenantUUID {
		_ = r.writeFileFunc(tenantFile, []byte(ci.TenantUUID), 0644)
	}

	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query OneAgent version: %w", err)
	}

	var oldVer string
	if b, err := r.readFileFunc(verFile); err != nil && !os.IsNotExist(err) {
		return reconcile.Result{}, fmt.Errorf("failed to query installed OneAgent version: %w", err)
	} else {
		oldVer = string(b)
	}

	arch := dtclient.ArchX86
	if runtime.GOARCH == "arm64" {
		arch = dtclient.ArchARM
	}

	if ver != oldVer {
		for _, flavor := range []string{dtclient.FlavorDefault, dtclient.FlavorMUSL} {
			targetDir := filepath.Join(envDir, "bin", ver+"-"+flavor)

			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				installAgentCfg := newInstallAgentConfig(rlog, dtc, flavor, arch, targetDir)

				if err := installAgent(installAgentCfg); err != nil {
					if errDel := os.RemoveAll(targetDir); errDel != nil {
						rlog.Error(errDel, "failed to delete target directory", "path", targetDir)
					}

					return reconcile.Result{}, fmt.Errorf("failed to install agent: %w", err)
				}
			}
		}

		_ = r.writeFileFunc(verFile, []byte(ver), 0644)
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}
