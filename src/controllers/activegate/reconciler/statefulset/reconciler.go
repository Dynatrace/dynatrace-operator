package statefulset

import (
	"context"
	"hash/fnv"
	"reflect"
	"strconv"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/events"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/secrets"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
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
	Instance                         *dynatracev1beta1.DynaKube
	apiReader                        client.Reader
	scheme                           *runtime.Scheme
	feature                          string
	capabilityName                   string
	serviceAccountOwner              string
	capability                       *dynatracev1beta1.CapabilityProperties
	onAfterStatefulSetCreateListener []events.StatefulSetEvent
	initContainersTemplates          []corev1.Container
	containerVolumeMounts            []corev1.VolumeMount
	volumes                          []corev1.Volume
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme,
	instance *dynatracev1beta1.DynaKube, capability capability.Capability) *Reconciler {

	serviceAccountOwner := capability.Config().ServiceAccountOwner
	if serviceAccountOwner == "" {
		serviceAccountOwner = capability.ShortName()
	}

	return &Reconciler{
		Client:                           clt,
		apiReader:                        apiReader,
		scheme:                           scheme,
		Instance:                         instance,
		feature:                          capability.ShortName(),
		capabilityName:                   capability.ArgName(),
		serviceAccountOwner:              serviceAccountOwner,
		capability:                       capability.Properties(),
		onAfterStatefulSetCreateListener: []events.StatefulSetEvent{},
		initContainersTemplates:          capability.InitContainersTemplates(),
		containerVolumeMounts:            capability.ContainerVolumeMounts(),
		volumes:                          capability.Volumes(),
	}
}

func (r *Reconciler) AddOnAfterStatefulSetCreateListener(event events.StatefulSetEvent) {
	r.onAfterStatefulSetCreateListener = append(r.onAfterStatefulSetCreateListener, event)
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if r.capability.CustomProperties != nil {
		err = customproperties.
			NewReconciler(r, r.Instance, r.serviceAccountOwner, *r.capability.CustomProperties, r.scheme).
			Reconcile()
		if err != nil {
			log.Error(err, "could not reconcile custom properties")
			return false, errors.WithStack(err)
		}
	}

	if update, err = r.manageStatefulSet(); err != nil {
		log.Error(err, "could not reconcile stateful set")
		return false, errors.WithStack(err)
	}

	return update, nil
}

func (r *Reconciler) manageStatefulSet() (bool, error) {
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

	deleted, err := r.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	if deleted || err != nil {
		return deleted, errors.WithStack(err)
	}

	updated, err := r.updateStatefulSetIfOutdated(desiredSts)
	if updated || err != nil {
		return updated, errors.WithStack(err)
	}

	return false, nil
}

func (r *Reconciler) buildDesiredStatefulSet() (*appsv1.StatefulSet, error) {
	kubeUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	agHash, err := r.calculateActiveGateConfigurationHash()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stsProperties := NewStatefulSetProperties(
		r.Instance, r.capability, kubeUID, agHash, r.feature, r.capabilityName, r.serviceAccountOwner,
		r.initContainersTemplates, r.containerVolumeMounts, r.volumes)
	stsProperties.OnAfterCreateListener = r.onAfterStatefulSetCreateListener

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
		log.Info("creating new stateful set for " + r.feature)
		return true, r.Create(context.TODO(), desiredSts)
	}
	return false, err
}

func (r *Reconciler) updateStatefulSetIfOutdated(desiredSts *appsv1.StatefulSet) (bool, error) {
	currentSts, err := r.getStatefulSet(desiredSts)
	if err != nil {
		return false, err
	}
	if !kubeobjects.HasChanged(currentSts, desiredSts) {
		return false, nil
	}

	if kubeobjects.LabelsNotEqual(currentSts.Spec.Selector.MatchLabels, desiredSts.Spec.Selector.MatchLabels) {
		return r.recreateStatefulSet(currentSts, desiredSts)
	}

	log.Info("updating existing stateful set")
	if err = r.Update(context.TODO(), desiredSts); err != nil {
		return false, err
	}
	return true, err
}

func (r *Reconciler) recreateStatefulSet(currentSts, desiredSts *appsv1.StatefulSet) (bool, error) {
	log.Info("immutable section changed on statefulset, deleting and recreating", "name", desiredSts.Name)

	err := r.Delete(context.TODO(), currentSts)
	if err != nil {
		return false, err
	}

	log.Info("deleted statefulset")
	log.Info("recreating statefulset", "name", desiredSts.Name)

	return true, r.Create(context.TODO(), desiredSts)
}

func (r *Reconciler) deleteStatefulSetIfOldLabelsAreUsed(desiredSts *appsv1.StatefulSet) (bool, error) {
	currentSts, err := r.getStatefulSet(desiredSts)
	if err != nil {
		return false, err
	}

	if !reflect.DeepEqual(currentSts.Labels, desiredSts.Labels) {
		log.Info("deleting existing stateful set")
		if err = r.Delete(context.TODO(), desiredSts); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (r *Reconciler) calculateActiveGateConfigurationHash() (string, error) {
	customPropData, err := r.getCustomPropertyHash()
	if err != nil {
		return "", errors.WithStack(err)
	}

	authTokenData, err := r.getAuthTokenHash()
	if err != nil {
		return "", errors.WithStack(err)
	}

	if len(customPropData) < 1 && len(authTokenData) < 1 {
		return "", nil
	}

	hash := fnv.New32()
	if _, err := hash.Write([]byte(customPropData + authTokenData)); err != nil {
		return "", errors.WithStack(err)
	}

	return strconv.FormatUint(uint64(hash.Sum32()), 10), nil
}

func (r *Reconciler) getCustomPropertyHash() (string, error) {
	if !r.needsCustomPropertyHash(r.capability.CustomProperties) {
		return "", nil
	}

	customPropData, err := r.getDataFromCustomProperty(r.capability.CustomProperties)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return customPropData, nil
}

func (r *Reconciler) getAuthTokenHash() (string, error) {
	if !r.Instance.UseActiveGateAuthToken() {
		return "", nil
	}

	authTokenData, err := r.getDataFromAuthTokenSecret()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return authTokenData, nil
}

func (r *Reconciler) getDataFromCustomProperty(customProperties *dynatracev1beta1.DynaKubeValueSource) (string, error) {
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

func (r *Reconciler) getDataFromAuthTokenSecret() (string, error) {
	namespace := r.Instance.Namespace
	var secret corev1.Secret
	err := r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.ActiveGateAuthTokenSecret(), Namespace: namespace}, &secret)
	if err != nil {
		return "", errors.WithStack(err)
	}

	dataBytes, ok := secret.Data[secrets.ActiveGateAuthTokenName]
	if !ok {
		return "", errors.Errorf("no auth-token found on secret '%s' on namespace '%s'", r.Instance.ActiveGateAuthTokenSecret(), namespace)
	}
	return string(dataBytes), nil
}

func (r *Reconciler) needsCustomPropertyHash(customProperties *dynatracev1beta1.DynaKubeValueSource) bool {
	return customProperties != nil && (customProperties.Value != "" || customProperties.ValueFrom != "")
}
