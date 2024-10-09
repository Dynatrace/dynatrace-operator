package statefulset

import (
	"hash/fnv"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ controllers.Reconciler = &Reconciler{}

type Reconciler struct {
	client     client.Client
	dk         *dynakube.DynaKube
	apiReader  client.Reader
	capability capability.Capability
	modifiers  []builder.Modifier
}

func NewReconciler(
	clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube,
	capability capability.Capability,
) controllers.Reconciler {
	return &Reconciler{
		client:     clt,
		apiReader:  apiReader,
		dk:         dk,
		capability: capability,
		modifiers:  []builder.Modifier{},
	}
}

type NewReconcilerFunc = func(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube, capability capability.Capability) controllers.Reconciler

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.manageStatefulSet(ctx)
	if err != nil {
		log.Error(err, "could not reconcile stateful set")

		return err
	}

	return nil
}

func (r *Reconciler) manageStatefulSet(ctx context.Context) error {
	desiredSts, err := r.buildDesiredStatefulSet(ctx)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), ActiveGateStatefulSetConditionType, err)

		return err
	}

	updated, err := statefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, desiredSts)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), ActiveGateStatefulSetConditionType, err)

		return err
	} else if updated {
		conditions.SetStatefulSetCreated(r.dk.Conditions(), ActiveGateStatefulSetConditionType, desiredSts.Name)
	}

	return nil
}

func (r *Reconciler) buildDesiredStatefulSet(ctx context.Context) (*appsv1.StatefulSet, error) {
	kubeUID := types.UID(r.dk.Status.KubeSystemUUID)

	activeGateConfigurationHash, err := r.calculateActiveGateConfigurationHash(ctx)
	if err != nil {
		return nil, err
	}

	statefulSetBuilder := NewStatefulSetBuilder(kubeUID, activeGateConfigurationHash, *r.dk, r.capability)

	desiredSts, err := statefulSetBuilder.CreateStatefulSet(r.modifiers)

	return desiredSts, err
}

func (r *Reconciler) calculateActiveGateConfigurationHash(ctx context.Context) (string, error) {
	customPropertyData, err := r.getCustomPropertyValue(ctx)
	if err != nil {
		return "", err
	}

	authTokenData, err := r.getAuthTokenValue(ctx)
	if err != nil {
		return "", err
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

func (r *Reconciler) getCustomPropertyValue(ctx context.Context) (string, error) {
	if !needsCustomPropertyHash(r.capability.Properties().CustomProperties) {
		return "", nil
	}

	customPropertyData, err := r.getDataFromCustomProperty(ctx, r.capability.Properties().CustomProperties)
	if err != nil {
		return "", err
	}

	return customPropertyData, nil
}

func (r *Reconciler) getAuthTokenValue(ctx context.Context) (string, error) {
	if !r.dk.ActiveGate().IsEnabled() {
		return "", nil
	}

	authTokenData, err := r.getDataFromAuthTokenSecret(ctx)
	if err != nil {
		return "", err
	}

	return authTokenData, nil
}

func (r *Reconciler) getDataFromCustomProperty(ctx context.Context, customProperties *value.Source) (string, error) {
	if customProperties.ValueFrom != "" {
		return secret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{Namespace: r.dk.Namespace, Name: customProperties.ValueFrom}, customproperties.DataKey, log)
	}

	return customProperties.Value, nil
}

func (r *Reconciler) getDataFromAuthTokenSecret(ctx context.Context) (string, error) {
	return secret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{Namespace: r.dk.Namespace, Name: r.dk.ActiveGate().GetAuthTokenSecretName()}, authtoken.ActiveGateAuthTokenName, log)
}

func needsCustomPropertyHash(customProperties *value.Source) bool {
	return customProperties != nil && (customProperties.Value != "" || customProperties.ValueFrom != "")
}
