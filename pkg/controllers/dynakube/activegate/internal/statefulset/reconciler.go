package statefulset

import (
	"hash/fnv"
	"reflect"
	"strconv"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	kubeobjects2 "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ controllers.Reconciler = &Reconciler{}

type Reconciler struct {
	client     client.Client
	dynakube   *dynatracev1beta1.DynaKube
	apiReader  client.Reader
	scheme     *runtime.Scheme
	capability capability.Capability
	modifiers  []builder.Modifier
}

// argument-limit doesn't apply to constructors
//
//nolint:revive
func NewReconciler(
	clt client.Client,
	apiReader client.Reader,
	scheme *runtime.Scheme,
	dynakube *dynatracev1beta1.DynaKube,
	capability capability.Capability,
) *Reconciler {
	return &Reconciler{
		client:     clt,
		apiReader:  apiReader,
		scheme:     scheme,
		dynakube:   dynakube,
		capability: capability,
		modifiers:  []builder.Modifier{},
	}
}

type NewReconcilerFunc = func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, capability capability.Capability) *Reconciler

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.manageStatefulSet(ctx)
	if err != nil {
		log.Error(err, "could not reconcile stateful set")
		return errors.WithStack(err)
	}

	return nil
}

func (r *Reconciler) manageStatefulSet(ctx context.Context) error {
	desiredSts, err := r.buildDesiredStatefulSet()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := controllerutil.SetControllerReference(r.dynakube, desiredSts, r.scheme); err != nil {
		return errors.WithStack(err)
	}

	created, err := r.createStatefulSetIfNotExists(ctx, desiredSts)
	if created || err != nil {
		return errors.WithStack(err)
	}

	deleted, err := r.deleteStatefulSetIfSelectorChanged(ctx, desiredSts)
	if deleted || err != nil {
		return errors.WithStack(err)
	}

	updated, err := r.updateStatefulSetIfOutdated(ctx, desiredSts)
	if updated || err != nil {
		return errors.WithStack(err)
	}

	return nil
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

	statefulSetBuilder := NewStatefulSetBuilder(kubeUID, activeGateConfigurationHash, *r.dynakube, r.capability)

	desiredSts, err := statefulSetBuilder.CreateStatefulSet(r.modifiers)
	return desiredSts, errors.WithStack(err)
}

func (r *Reconciler) getStatefulSet(ctx context.Context, desiredSts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	var sts appsv1.StatefulSet
	err := r.client.Get(ctx, client.ObjectKey{Name: desiredSts.Name, Namespace: desiredSts.Namespace}, &sts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &sts, nil
}

func (r *Reconciler) createStatefulSetIfNotExists(ctx context.Context, desiredSts *appsv1.StatefulSet) (bool, error) {
	_, err := r.getStatefulSet(ctx, desiredSts)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		log.Info("creating new stateful set for " + r.capability.ShortName())
		return true, r.client.Create(ctx, desiredSts)
	}
	return false, err
}

func (r *Reconciler) updateStatefulSetIfOutdated(ctx context.Context, desiredSts *appsv1.StatefulSet) (bool, error) {
	currentSts, err := r.getStatefulSet(ctx, desiredSts)
	if err != nil {
		return false, err
	}
	if !kubeobjects2.IsHashAnnotationDifferent(currentSts, desiredSts) {
		return false, nil
	}

	if kubeobjects2.LabelsNotEqual(currentSts.Spec.Selector.MatchLabels, desiredSts.Spec.Selector.MatchLabels) {
		return r.recreateStatefulSet(ctx, currentSts, desiredSts)
	}

	log.Info("updating existing stateful set")
	if err = r.client.Update(ctx, desiredSts); err != nil {
		return false, err
	}
	return true, err
}

func (r *Reconciler) recreateStatefulSet(ctx context.Context, currentSts, desiredSts *appsv1.StatefulSet) (bool, error) {
	log.Info("immutable section changed on statefulset, deleting and recreating", "name", desiredSts.Name)

	err := r.client.Delete(ctx, currentSts)
	if err != nil {
		return false, err
	}

	log.Info("deleted statefulset")
	log.Info("recreating statefulset", "name", desiredSts.Name)

	return true, r.client.Create(ctx, desiredSts)
}

// the selector, e.g. MatchLabels, of a stateful set is immutable.
// if it changed, for example due to a new operator version, deleteStatefulSetIfSelectorChanged deletes the stateful set
// so it can be updated correctly afterwards.
func (r *Reconciler) deleteStatefulSetIfSelectorChanged(ctx context.Context, desiredSts *appsv1.StatefulSet) (bool, error) {
	currentSts, err := r.getStatefulSet(ctx, desiredSts)
	if err != nil {
		return false, err
	}

	if hasSelectorChanged(desiredSts, currentSts) {
		log.Info("deleting existing stateful set because selector changed")
		if err = r.client.Delete(ctx, desiredSts); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func hasSelectorChanged(desiredSts *appsv1.StatefulSet, currentSts *appsv1.StatefulSet) bool {
	return !reflect.DeepEqual(currentSts.Spec.Selector, desiredSts.Spec.Selector)
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
	if !r.dynakube.UseActiveGateAuthToken() {
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
		return kubeobjects2.GetDataFromSecretName(r.apiReader, types.NamespacedName{Namespace: r.dynakube.Namespace, Name: customProperties.ValueFrom}, customproperties.DataKey, log)
	}
	return customProperties.Value, nil
}

func (r *Reconciler) getDataFromAuthTokenSecret() (string, error) {
	return kubeobjects2.GetDataFromSecretName(r.apiReader, types.NamespacedName{Namespace: r.dynakube.Namespace, Name: r.dynakube.ActiveGateAuthTokenSecret()}, authtoken.ActiveGateAuthTokenName, log)
}

func needsCustomPropertyHash(customProperties *dynatracev1beta1.DynaKubeValueSource) bool {
	return customProperties != nil && (customProperties.Value != "" || customProperties.ValueFrom != "")
}
