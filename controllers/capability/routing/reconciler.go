package routing

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/customproperties"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"hash/fnv"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

const (
	module            = "msgrouter"
	StatefulSetSuffix = "-" + module
	CapabilityEnv     = "MSGrouter"
)

type ReconcileRouting struct {
	client.Client
	apiReader            client.Reader
	scheme               *runtime.Scheme
	dtc                  dtclient.Client
	log                  logr.Logger
	instance             *v1alpha1.DynaKube
	imageVersionProvider dtversion.ImageVersionProvider
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *v1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider) *ReconcileRouting {
	return &ReconcileRouting{
		Client:               clt,
		apiReader:            apiReader,
		scheme:               scheme,
		dtc:                  dtc,
		log:                  log,
		instance:             instance,
		imageVersionProvider: imageVersionProvider,
	}
}

func (r *ReconcileRouting) Reconcile() (update bool, err error) {
	if r.instance.Spec.RoutingSpec.CustomProperties != nil {
		err = customproperties.
			NewReconciler(r, r.instance, r.log, module, *r.instance.Spec.RoutingSpec.CustomProperties, r.scheme).
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

func (r *ReconcileRouting) manageStatefulSet() (bool, error) {
	desiredSts, err := r.buildDesiredStatefulSet()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if err := controllerutil.SetControllerReference(r.instance, desiredSts, r.scheme); err != nil {
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

func (r *ReconcileRouting) buildDesiredStatefulSet() (*appsv1.StatefulSet, error) {
	kubeUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cpHash, err := r.getCustomPropsHash()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	desiredSts, err := capability.CreateStatefulSet(
		capability.NewStatefulSetProperties(
			r.instance, &r.instance.Spec.RoutingSpec.CapabilityProperties, kubeUID, cpHash, module, CapabilityEnv, ""))
	return desiredSts, errors.WithStack(err)
}

func (r *ReconcileRouting) getCustomPropsHash() (string, error) {
	return "", nil
}

func (r *ReconcileRouting) getStatefulSet(desiredSts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	var sts appsv1.StatefulSet
	err := r.Get(context.TODO(), client.ObjectKey{Name: desiredSts.Name, Namespace: desiredSts.Namespace}, &sts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &sts, nil
}

func (r *ReconcileRouting) createStatefulSetIfNotExists(desiredSts *appsv1.StatefulSet) (bool, error) {
	_, err := r.getStatefulSet(desiredSts)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		r.log.Info("creating new stateful set for " + module)
		return true, r.Create(context.TODO(), desiredSts)
	}
	return false, err
}

func (r *ReconcileRouting) updateStatefulSetIfOutdated(desiredSts *appsv1.StatefulSet) (bool, error) {
	currentSts, err := r.getStatefulSet(desiredSts)
	if err != nil {
		return false, err
	}
	if !capability.HasStatefulSetChanged(currentSts, desiredSts) {
		return false, nil
	}

	r.log.Info("updating existing stateful set")
	if err = r.Update(context.TODO(), desiredSts); err != nil {
		return false, err
	}
	return true, err
}

func (r *ReconcileRouting) calculateCustomPropertyHash() (string, error) {
	customProperties := r.instance.Spec.RoutingSpec.CustomProperties
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

func (r *ReconcileRouting) getDataFromCustomProperty(customProperties *v1alpha1.DynaKubeValueSource) (string, error) {
	if customProperties.ValueFrom != "" {
		namespace := r.instance.Namespace
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
