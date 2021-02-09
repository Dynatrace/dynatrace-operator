package capability

import (
	"context"
	"hash/fnv"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/customproperties"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client.Client
	Instance                 *v1alpha1.DynaKube
	apiReader                client.Reader
	scheme                   *runtime.Scheme
	dtc                      dtclient.Client
	log                      logr.Logger
	imageVersionProvider     dtversion.ImageVersionProvider
	enableUpdates            bool
	module                   string
	capabilityName           string
	serviceAccountOwner      string
	capability               *v1alpha1.CapabilityProperties
	onAfterStatefulSetCreate []StatefulSetEvent
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *v1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider, enableUpdates bool,
	capability *v1alpha1.CapabilityProperties, module string, capabilityName string, serviceAccountOwner string) *Reconciler {
	if serviceAccountOwner == "" {
		serviceAccountOwner = module
	}

	return &Reconciler{
		Client:                   clt,
		apiReader:                apiReader,
		scheme:                   scheme,
		dtc:                      dtc,
		log:                      log,
		Instance:                 instance,
		imageVersionProvider:     imageVersionProvider,
		enableUpdates:            enableUpdates,
		module:                   module,
		capabilityName:           capabilityName,
		serviceAccountOwner:      serviceAccountOwner,
		capability:               capability,
		onAfterStatefulSetCreate: []StatefulSetEvent{},
	}
}

func (r *Reconciler) AddOnAfterStatefulSetCreate(event StatefulSetEvent) {
	r.onAfterStatefulSetCreate = append(r.onAfterStatefulSetCreate, event)
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if r.capability.CustomProperties != nil {
		err = customproperties.
			NewReconciler(r, r.Instance, r.log, r.serviceAccountOwner, *r.capability.CustomProperties, r.scheme).
			Reconcile()
		if err != nil {
			r.log.Error(err, "could not reconcile custom properties")
			return false, errors.WithStack(err)
		}
	}

	if update, err = r.manageStatefulSet(); err != nil {
		r.log.Error(err, "could not reconcile stateful set")
		return false, errors.WithStack(err)
	}

	return update, nil
}

func (r *Reconciler) manageStatefulSet() (bool, error) {
	versionUpdated, err := r.updateImageVersion()
	if versionUpdated || err != nil {
		return versionUpdated, errors.WithStack(err)
	}

	desiredSts, err := r.buildDesiredStatefulSet()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if err := controllerutil.SetControllerReference(r.Instance, desiredSts, r.scheme); err != nil {
		return false, errors.WithStack(err)
	}

	created, err := r.createStatefulSetIfNotExists(desiredSts)
	if created || err != nil {
		return created, errors.WithStack(err)
	}

	updated, err := r.updateStatefulSetIfOutdated(desiredSts)
	if updated || err != nil {
		return updated, errors.WithStack(err)
	}

	return false, nil
}

func (r *Reconciler) updateImageVersion() (bool, error) {
	if !r.enableUpdates {
		return false, nil
	}

	img := utils.BuildActiveGateImage(r.Instance)
	instance := r.Instance
	pullSecret, err := dtpullsecret.GetImagePullSecret(r, instance)
	if err != nil {
		return false, errors.WithMessage(err, "failed to get image pull secret")
	}

	auths, err := dtversion.ParseDockerAuthsFromSecret(pullSecret)
	if err != nil {
		return false, errors.WithMessage(err, "failed to get Dockerconfig for pull secret")
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
		return false, errors.WithMessage(err, "failed to get image version")
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

func (r *Reconciler) buildDesiredStatefulSet() (*appsv1.StatefulSet, error) {
	kubeUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cpHash, err := r.calculateCustomPropertyHash()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stsProperties := NewStatefulSetProperties(
		r.Instance, r.capability, kubeUID, cpHash, r.module, r.capabilityName, r.serviceAccountOwner)
	stsProperties.onAfterCreate = r.onAfterStatefulSetCreate

	desiredSts, err := CreateStatefulSet(stsProperties)
	return desiredSts, errors.WithStack(err)
}

func (r *Reconciler) getStatefulSet(desiredSts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	var sts appsv1.StatefulSet
	err := r.Get(context.TODO(), client.ObjectKey{Name: desiredSts.Name, Namespace: desiredSts.Namespace}, &sts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &sts, nil
}

func (r *Reconciler) createStatefulSetIfNotExists(desiredSts *appsv1.StatefulSet) (bool, error) {
	_, err := r.getStatefulSet(desiredSts)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		r.log.Info("creating new stateful set for " + r.module)
		return true, r.Create(context.TODO(), desiredSts)
	}
	return false, err
}

func (r *Reconciler) updateStatefulSetIfOutdated(desiredSts *appsv1.StatefulSet) (bool, error) {
	currentSts, err := r.getStatefulSet(desiredSts)
	if err != nil {
		return false, err
	}
	if !HasStatefulSetChanged(currentSts, desiredSts) {
		return false, nil
	}

	r.log.Info("updating existing stateful set")
	if err = r.Update(context.TODO(), desiredSts); err != nil {
		return false, err
	}
	return true, err
}

func (r *Reconciler) calculateCustomPropertyHash() (string, error) {
	customProperties := r.capability.CustomProperties
	if customProperties == nil || (customProperties.Value == "" && customProperties.ValueFrom == "") {
		return "", nil
	}

	data, err := r.getDataFromCustomProperty(customProperties)
	if err != nil {
		return "", errors.WithStack(err)
	}

	hash := fnv.New32()
	if _, err = hash.Write([]byte(data)); err != nil {
		return "", errors.WithStack(err)
	}

	return strconv.FormatUint(uint64(hash.Sum32()), 10), nil
}

func (r *Reconciler) getDataFromCustomProperty(customProperties *v1alpha1.DynaKubeValueSource) (string, error) {
	if customProperties.ValueFrom != "" {
		namespace := r.Instance.Namespace
		var secret corev1.Secret
		err := r.Get(context.TODO(), client.ObjectKey{Name: customProperties.ValueFrom, Namespace: namespace}, &secret)
		if err != nil {
			return "", errors.WithStack(err)
		}

		dataBytes, ok := secret.Data[customproperties.DataKey]
		if !ok {
			return "", errors.Errorf("no custom properties found on secret '%s' on namespace '%s'", customProperties.ValueFrom, namespace)
		}
		return string(dataBytes), nil
	}
	return customProperties.Value, nil
}
