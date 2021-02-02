package kubemon

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
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
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	Name = "kubernetes-monitoring"

	annotationTemplateHash    = "internal.operator.dynatrace.com/template-hash"
	annotationImageHash       = "internal.operator.dynatrace.com/image-hash"
	annotationImageVersion    = "internal.operator.dynatrace.com/image-version"
	annotationCustomPropsHash = "internal.operator.dynatrace.com/custom-properties-hash"

	envVarDisableUpdates = "OPERATOR_DEBUG_DISABLE_UPDATES"
)

type ReconcileKubeMon struct {
	client.Client
	scheme    *runtime.Scheme
	dtc       dtclient.Client
	log       logr.Logger
	instance  *dynatracev1alpha1.DynaKube
	apiReader client.Reader

	imageVersionProvider dtversion.ImageVersionProvider
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *dynatracev1alpha1.DynaKube, imgVerProvider dtversion.ImageVersionProvider) *ReconcileKubeMon {
	return &ReconcileKubeMon{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		dtc:       dtc,
		log:       log,
		instance:  instance,

		imageVersionProvider: imgVerProvider,
	}
}

func (r *ReconcileKubeMon) Reconcile() (update bool, err error) {
	if r.instance.Spec.KubernetesMonitoringSpec.CustomProperties != nil {
		err = customproperties.
			NewReconciler(r, r.instance, r.log, Name, *r.instance.Spec.KubernetesMonitoringSpec.CustomProperties, r.scheme).
			Reconcile()
		if err != nil {
			r.log.Error(err, "could not reconcile custom properties")
			return false, err
		}
	}

	if update, err = r.manageStatefulSet(r.instance); err != nil {
		r.log.Error(err, "could not reconcile stateful set")
		return false, err
	}

	return update, nil
}

func (r *ReconcileKubeMon) manageStatefulSet(instance *dynatracev1alpha1.DynaKube) (update bool, err error) {
	if os.Getenv(envVarDisableUpdates) != "true" {
		img := utils.BuildActiveGateImage(instance)
		if update, err = r.updateImageVersion(instance, img); err != nil {
			r.log.Error(err, "Failed to fetch image version", "image", img)
		}
	}

	desiredStatefulSet, err := r.buildDesiredStatefulSet(instance)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if err := controllerutil.SetControllerReference(instance, desiredStatefulSet, r.scheme); err != nil {
		return false, errors.WithStack(err)
	}

	currentStatefulSet, stsCreated, err := r.createStatefulSetIfNotExists(desiredStatefulSet)
	if err != nil {
		return false, errors.WithStack(err)
	}

	stsChanged, err := r.updateStatefulSetIfOutdated(currentStatefulSet, desiredStatefulSet)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if !update && !stsCreated && !stsChanged {
		return false, nil
	}

	return true, nil
}

func (r *ReconcileKubeMon) updateImageVersion(instance *dynatracev1alpha1.DynaKube, img string) (bool, error) {
	pullSecret, err := dtpullsecret.GetImagePullSecret(r, r.instance)
	if err != nil {
		return false, fmt.Errorf("failed to get image pull secret: %w", err)
	}

	auths, err := dtversion.ParseDockerAuthsFromSecret(pullSecret)
	if err != nil {
		return false, fmt.Errorf("failed to get Dockerconfig for pull secret: %w", err)
	}

	verProvider := dtversion.GetImageVersion
	if r.imageVersionProvider != nil {
		verProvider = r.imageVersionProvider
	}

	ver, err := verProvider(img, &dtversion.DockerConfig{
		Auths:         auths,
		SkipCertCheck: instance.Spec.SkipCertCheck,
	})
	if err != nil {
		return false, fmt.Errorf("failed to get image version: %w", err)
	}

	upd := false
	if instance.Status.ActiveGate.ImageHash != ver.Hash {
		r.log.Info("Update found",
			"oldVersion", instance.Status.ActiveGate.ImageVersion,
			"newVersion", ver.Version,
			"oldHash", instance.Status.ActiveGate.ImageHash,
			"newHash", ver.Hash)
		upd = true
	}

	instance.Status.ActiveGate.ImageVersion = ver.Version
	instance.Status.ActiveGate.ImageHash = ver.Hash
	return upd, nil
}

func (r *ReconcileKubeMon) buildDesiredStatefulSet(instance *dynatracev1alpha1.DynaKube) (*appsv1.StatefulSet, error) {
	kubeUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cpHash, err := r.getCustomPropsHash()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return newStatefulSet(instance, kubeUID, cpHash)
}

func (r *ReconcileKubeMon) getCustomPropsHash() (string, error) {
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
			return "", fmt.Errorf("no custom properties found on secret '%s' on namespace '%s'", cp.ValueFrom, ns)
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

func (r *ReconcileKubeMon) createStatefulSetIfNotExists(desired *appsv1.StatefulSet) (*appsv1.StatefulSet, bool, error) {
	currentStatefulSet, err := r.getCurrentStatefulSet(desired)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		r.log.Info("creating new stateful set for kubernetes monitoring")
		return desired, true, r.createStatefulSet(desired)
	}
	return currentStatefulSet, false, err
}

func (r *ReconcileKubeMon) updateStatefulSetIfOutdated(current *appsv1.StatefulSet, desired *appsv1.StatefulSet) (bool, error) {
	if !hasStatefulSetChanged(current, desired) {
		return false, nil
	}

	r.log.Info("updating existing stateful set")
	if err := r.Update(context.TODO(), desired); err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (r *ReconcileKubeMon) getCurrentStatefulSet(desired *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	var currentStatefulSet appsv1.StatefulSet
	err := r.Get(context.TODO(), client.ObjectKey{Name: desired.Name, Namespace: desired.Namespace}, &currentStatefulSet)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &currentStatefulSet, nil
}

func (r *ReconcileKubeMon) createStatefulSet(desired *appsv1.StatefulSet) error {
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
