package core

import (
	"context"
	"hash/fnv"
	"reflect"
	"strconv"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/secrets"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client.Client
	Dynakube                         *dynatracev1beta1.DynaKube
	apiReader                        client.Reader
	scheme                           *runtime.Scheme
	serviceAccountOwner              string
	onAfterStatefulSetCreateListener []statefulset.StatefulSetEvent
	capability                       capability.Capability
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme,
	dynakube *dynatracev1beta1.DynaKube, capability capability.Capability) *Reconciler {

	serviceAccountOwner := capability.Config().ServiceAccountOwner
	if serviceAccountOwner == "" {
		serviceAccountOwner = capability.ShortName()
	}

	return &Reconciler{
		Client:                           clt,
		apiReader:                        apiReader,
		scheme:                           scheme,
		Dynakube:                         dynakube,
		capability:                       capability,
		serviceAccountOwner:              serviceAccountOwner,
		onAfterStatefulSetCreateListener: []statefulset.StatefulSetEvent{},
	}
}

func (r *Reconciler) AddOnAfterStatefulSetCreateListener(event statefulset.StatefulSetEvent) {
	r.onAfterStatefulSetCreateListener = append(r.onAfterStatefulSetCreateListener, event)
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if r.capability.Properties().CustomProperties != nil {
		err = customproperties.NewReconciler(r, r.Dynakube, r.serviceAccountOwner, *r.capability.Properties().CustomProperties, r.scheme).
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

	if err := controllerutil.SetControllerReference(r.Dynakube, desiredSts, r.scheme); err != nil {
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

	activeGateConfigurationHash, err := r.calculateActiveGateConfigurationHash()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stsProperties := statefulset.NewStatefulSetProperties(
		r.Dynakube, r.capability.Properties(), kubeUID, activeGateConfigurationHash, r.capability.ShortName(), r.capability.ArgName(), r.serviceAccountOwner,
		r.capability.InitContainersTemplates(), r.capability.ContainerVolumeMounts(), r.capability.Volumes())
	stsProperties.OnAfterCreateListener = r.onAfterStatefulSetCreateListener

	desiredSts, err := statefulset.CreateStatefulSet(stsProperties)
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
		log.Info("creating new stateful set for " + r.capability.ShortName())
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
	customPropertyData, err := r.getCustomPropertyValue()
	if err != nil {
		return "", errors.WithStack(err)
	}

	authTokenData, err := r.getAuthTokenValue()
	if err != nil {
		return "", errors.WithStack(err)
	}

	if len(customPropertyData) < 1 && len(authTokenData) < 1 {
		return "", nil
	}

	hash := fnv.New32()
	if _, err := hash.Write([]byte(customPropertyData + authTokenData)); err != nil {
		return "", errors.WithStack(err)
	}

	return strconv.FormatUint(uint64(hash.Sum32()), 10), nil
}

func (r *Reconciler) getCustomPropertyValue() (string, error) {
	if !needsCustomPropertyHash(r.capability.Properties().CustomProperties) {
		return "", nil
	}

	customPropertyData, err := r.getDataFromCustomProperty(r.capability.Properties().CustomProperties)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return customPropertyData, nil
}

func (r *Reconciler) getAuthTokenValue() (string, error) {
	if !r.Dynakube.UseActiveGateAuthToken() {
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
		return kubeobjects.GetDataFromSecretName(r.apiReader, types.NamespacedName{Namespace: r.Dynakube.Namespace, Name: customProperties.ValueFrom}, customproperties.DataKey)
	}
	return customProperties.Value, nil
}

func (r *Reconciler) getDataFromAuthTokenSecret() (string, error) {
	return kubeobjects.GetDataFromSecretName(r.apiReader, types.NamespacedName{Namespace: r.Dynakube.Namespace, Name: r.Dynakube.ActiveGateAuthTokenSecret()}, secrets.ActiveGateAuthTokenName)
}

func needsCustomPropertyHash(customProperties *dynatracev1beta1.DynaKubeValueSource) bool {
	return customProperties != nil && (customProperties.Value != "" || customProperties.ValueFrom != "")
}
