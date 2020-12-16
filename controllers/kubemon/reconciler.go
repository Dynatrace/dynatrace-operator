package kubemon

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/customproperties"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	Name = "kubernetes-monitoring"

	annotationTemplateHash    = "internal.operator.dynatrace.com/template-hash"
	annotationImageHash       = "internal.operator.dynatrace.com/image-hash"
	annotationImageVersion    = "internal.operator.dynatrace.com/image-version"
	annotationCustomPropsHash = "internal.operator.dynatrace.com/custom-properties-hash"

	envVarDisableUpdates = "OPERATOR_DEBUG_DISABLE_UPDATES"
)

type Reconciler struct {
	client.Client
	scheme    *runtime.Scheme
	dtc       dtclient.Client
	log       logr.Logger
	token     *corev1.Secret
	instance  *dynatracev1alpha1.DynaKube
	apiReader client.Reader

	imageVersionProvider dtversion.ImageVersionProvider
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger, token *corev1.Secret,
	instance *dynatracev1alpha1.DynaKube, imgVerProvider dtversion.ImageVersionProvider) *Reconciler {
	return &Reconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		dtc:       dtc,
		log:       log,
		token:     token,
		instance:  instance,

		imageVersionProvider: imgVerProvider,
	}
}

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	err := dtpullsecret.
		NewReconciler(r, r.apiReader, r.scheme, r.instance, r.dtc, r.log, r.token, r.instance.Spec.ActiveGate.Image).
		Reconcile()
	if err != nil {
		r.log.Error(err, "could not reconcile Dynatrace pull secret")
		return reconcile.Result{}, err
	}

	if r.instance.Spec.KubernetesMonitoringSpec.CustomProperties != nil {
		err = customproperties.
			NewReconciler(r, r.instance, r.log, Name, *r.instance.Spec.KubernetesMonitoringSpec.CustomProperties, r.scheme).
			Reconcile()
		if err != nil {
			r.log.Error(err, "could not reconcile custom properties")
			return reconcile.Result{}, err
		}
	}

	if err = r.manageStatefulSet(r.instance); err != nil {
		r.log.Error(err, "could not reconcile stateful set")
		return reconcile.Result{}, err
	}

	if r.instance.Spec.KubernetesMonitoringSpec.KubernetesAPIEndpoint != "" {
		id, err := r.addToDashboard()
		r.handleAddToDashboardResult(id, err, r.log)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) manageStatefulSet(instance *dynatracev1alpha1.DynaKube) error {
	var err error

	verUpd := false
	if os.Getenv(envVarDisableUpdates) != "true" {
		img := buildImage(instance)
		if verUpd, err = r.updateImageVersion(instance, img); err != nil {
			r.log.Error(err, "Failed to fetch image version", "image", img)
		}
	}

	desiredStatefulSet, err := r.buildDesiredStatefulSet(instance)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(instance, desiredStatefulSet, r.scheme); err != nil {
		return err
	}

	currentStatefulSet, stsCreated, err := r.createStatefulSetIfNotExists(desiredStatefulSet)
	if err != nil {
		return err
	}

	stsChanged, err := r.updateStatefulSetIfOutdated(currentStatefulSet, desiredStatefulSet)
	if err != nil {
		return err
	}

	if !verUpd && !stsCreated && !stsChanged {
		return nil
	}

	return r.updateInstanceStatus(instance)
}

func (r *Reconciler) updateImageVersion(instance *dynatracev1alpha1.DynaKube, img string) (bool, error) {
	pullSecret, err := dtpullsecret.GetImagePullSecret(r, r.instance)
	if err != nil {
		return false, fmt.Errorf("failed to get image pull secret: %w", err)
	}

	dockerCfg, err := dtversion.NewDockerConfig(pullSecret)
	if err != nil {
		return false, fmt.Errorf("failed to get Dockerconfig for pull secret: %w", err)
	}

	verProvider := dtversion.GetImageVersion
	if r.imageVersionProvider != nil {
		verProvider = r.imageVersionProvider
	}

	ver, err := verProvider(img, dockerCfg)
	if err != nil {
		return false, fmt.Errorf("failed to get image version: %w", err)
	}

	upd := false
	if instance.Status.ActiveGateImageHash != ver.Hash {
		r.log.Info("Update found",
			"oldVersion", instance.Status.ActiveGateImageVersion,
			"newVersion", ver.Version,
			"oldHash", instance.Status.ActiveGateImageHash,
			"newHash", ver.Hash)
		upd = true
	}

	instance.Status.ActiveGateImageVersion = ver.Version
	instance.Status.ActiveGateImageHash = ver.Hash
	return upd, nil
}

func (r *Reconciler) buildDesiredStatefulSet(instance *dynatracev1alpha1.DynaKube) (*appsv1.StatefulSet, error) {
	kubeUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return nil, err
	}

	cpHash, err := r.getCustomPropsHash()
	if err != nil {
		return nil, err
	}

	return newStatefulSet(instance, kubeUID, cpHash)
}

func (r *Reconciler) getCustomPropsHash() (string, error) {
	cp := r.instance.Spec.KubernetesMonitoringSpec.CustomProperties
	if cp == nil || (cp.Value == "" && cp.ValueFrom == "") {
		return "", nil
	}

	hasher := fnv.New32()
	data := ""

	if cp.ValueFrom != "" {
		ns := r.instance.Namespace

		var secret corev1.Secret
		if err := r.Get(context.TODO(), client.ObjectKey{Name: cp.ValueFrom, Namespace: ns}, &secret); err != nil {
			return "", err
		}

		dataBytes, ok := secret.Data[customproperties.DataKey]
		if !ok {
			return "", fmt.Errorf("No custom properties found on secret '%s' on namespace '%s'", cp.ValueFrom, ns)
		}

		data = string(dataBytes)
	} else {
		data = cp.Value
	}

	if _, err := hasher.Write([]byte(data)); err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func (r *Reconciler) createStatefulSetIfNotExists(desired *appsv1.StatefulSet) (*appsv1.StatefulSet, bool, error) {
	currentStatefulSet, err := r.getCurrentStatefulSet(desired)
	if err != nil && k8serrors.IsNotFound(err) {
		r.log.Info("creating new stateful set for kubernetes monitoring")
		return desired, true, r.createStatefulSet(desired)
	}
	return currentStatefulSet, false, err
}

func (r *Reconciler) updateStatefulSetIfOutdated(current *appsv1.StatefulSet, desired *appsv1.StatefulSet) (bool, error) {
	if !hasStatefulSetChanged(current, desired) {
		return false, nil
	}

	r.log.Info("updating existing stateful set")
	if err := r.Update(context.TODO(), desired); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) updateInstanceStatus(instance *dynatracev1alpha1.DynaKube) error {
	instance.Status.UpdatedTimestamp = metav1.Now()
	instance.Status.Tokens = r.token.Name
	return r.Status().Update(context.TODO(), instance)
}

func (r *Reconciler) getCurrentStatefulSet(desired *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	var currentStatefulSet appsv1.StatefulSet
	err := r.Get(context.TODO(), client.ObjectKey{Name: desired.Name, Namespace: desired.Namespace}, &currentStatefulSet)
	if err != nil {
		return nil, err
	}
	return &currentStatefulSet, nil
}

func (r *Reconciler) createStatefulSet(desired *appsv1.StatefulSet) error {
	return r.Create(context.TODO(), desired)
}

func hasStatefulSetChanged(a *appsv1.StatefulSet, b *appsv1.StatefulSet) bool {
	return getTemplateHash(a) != getTemplateHash(b)
}

func getTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[annotationTemplateHash]
	}
	return ""
}
