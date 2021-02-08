package csiprovisioner

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logger.NewDTLogger().WithName("provisioner")

// NewReconciler returns a new OneAgentProvisioner
func NewReconciler(mgr manager.Manager, dataDir string) *OneAgentProvisioner {
	return &OneAgentProvisioner{
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		dtcBuildFunc: activegate.BuildDynatraceClient,
		dataDir:      dataDir,
	}
}

func (r *OneAgentProvisioner) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Complete(r)
}

var _ reconcile.Reconciler = &OneAgentProvisioner{}

// OneAgentProvisioner reconciles a DynaKube object
type OneAgentProvisioner struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	scheme       *k8sruntime.Scheme
	dtcBuildFunc activegate.DynatraceClientFunc
	dataDir      string
}

func (r *OneAgentProvisioner) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()

	rlog := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rlog.Info("Reconciling DynaKube")

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: activegate.GetTokensName(&dk), Namespace: dk.Namespace}, &tkns); err != nil {
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

	envDir := filepath.Join(r.dataDir, ci.TenantUUID)
	verFile := filepath.Join(envDir, "version")
	tenantFile := filepath.Join(r.dataDir, dk.Name)

	for _, dir := range []string{
		envDir,
		filepath.Join(envDir, "log"),
		filepath.Join(envDir, "datastorage"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	var oldDynaKubeTenant string
	if b, err := ioutil.ReadFile(tenantFile); err != nil && !os.IsNotExist(err) {
		return reconcile.Result{}, fmt.Errorf("failed to query assigned DynaKube tenant: %w", err)
	} else {
		oldDynaKubeTenant = string(b)
	}

	if oldDynaKubeTenant != ci.TenantUUID {
		ioutil.WriteFile(tenantFile, []byte(ci.TenantUUID), 0644)
	}

	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query OneAgent version: %w", err)
	}

	var oldVer string
	if b, err := ioutil.ReadFile(verFile); err != nil && !os.IsNotExist(err) {
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
			targetDir := filepath.Join(envDir, ver+"."+flavor)

			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				if err := installAgent(rlog, dtc, flavor, arch, targetDir); err != nil {
					if errDel := os.RemoveAll(targetDir); errDel != nil {
						rlog.Error(errDel, "failed to delete target directory", "path", targetDir)
					}

					return reconcile.Result{}, fmt.Errorf("failed to install agent: %w", err)
				}
			}
		}

		ioutil.WriteFile(verFile, []byte(ver), 0644)
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

func installAgent(rlog logr.Logger, dtc dtclient.Client, flavor, arch, targetDir string) error {
	tmpFile, err := ioutil.TempFile("", "download")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for download: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
		if err := os.Remove(tmpFile.Name()); err != nil {
			rlog.Error(err, "Failed to delete downloaded file", "path", tmpFile.Name())
		}
	}()

	rlog.Info("Downloading OneAgent package", "flavor", flavor, arch)

	r, err := dtc.GetLatestAgent(dtclient.OsUnix, dtclient.InstallerTypePaaS, flavor, arch)
	if err != nil {
		return fmt.Errorf("failed to fetch latest OneAgent version: %w", err)
	}
	defer r.Close()

	rlog.Info("Saving OneAgent package", "dest", tmpFile.Name())

	size, err := io.Copy(tmpFile, r)
	if err != nil {
		return fmt.Errorf("failed to save OneAgent package: %w", err)
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to save OneAgent package: %w", err)
	}

	zipr, err := zip.NewReader(tmpFile, size)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}

	rlog.Info("Unzipping OneAgent package")
	if err := unzip(rlog, zipr, targetDir); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}

	rlog.Info("Unzipped OneAgent package")

	for _, dir := range []string{
		filepath.Join(targetDir, "log"),
		filepath.Join(targetDir, "datastorage"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func unzip(rlog logr.Logger, r *zip.Reader, outDir string) error {
	os.MkdirAll(outDir, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extract := func(zipf *zip.File) error {
		rc, err := zipf.Open()
		if err != nil {
			return err
		}

		defer func() {
			if err := rc.Close(); err != nil {
				rlog.Error(err, "Failed to close ZIP entry file", "path", zipf.Name)
			}
		}()

		path := filepath.Join(outDir, zipf.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(outDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if zipf.FileInfo().IsDir() {
			return os.MkdirAll(path, zipf.Mode())
		}

		if err = os.MkdirAll(filepath.Dir(path), zipf.Mode()); err != nil {
			return err
		}

		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipf.Mode())
		if err != nil {
			return err
		}

		defer func() {
			if err := f.Close(); err != nil {
				rlog.Error(err, "Failed to close target file", "path", f.Name)
			}
		}()

		_, err = io.Copy(f, rc)
		return err
	}

	for _, f := range r.File {
		if err := extract(f); err != nil {
			return err
		}
	}

	return nil
}
