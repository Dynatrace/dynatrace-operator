package statefulset

import (
	"context"
	"hash/fnv"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	apiReader    client.Reader
	statefulsets k8sstatefulset.QueryObject
}

func NewReconciler(
	clt client.Client,
	apiReader client.Reader,
) *Reconciler {
	return &Reconciler{
		apiReader:    apiReader,
		statefulsets: k8sstatefulset.Query(clt, apiReader),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube, agCapability capability.Capability) error {
	err := r.manageStatefulSet(ctx, dk, agCapability)
	if err != nil {
		log.Error(err, "could not reconcile stateful set")

		return err
	}

	return nil
}

func (r *Reconciler) manageStatefulSet(ctx context.Context, dk *dynakube.DynaKube, agCapability capability.Capability) error {
	desiredSts, err := r.buildDesiredStatefulSet(ctx, dk, agCapability)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ActiveGateStatefulSetConditionType, err)

		return err
	}

	updated, err := r.statefulsets.WithOwner(dk).CreateOrUpdate(ctx, desiredSts)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ActiveGateStatefulSetConditionType, err)

		return err
	} else if updated {
		k8sconditions.SetStatefulSetCreated(dk.Conditions(), ActiveGateStatefulSetConditionType, desiredSts.Name)
	}

	return nil
}

func (r *Reconciler) buildDesiredStatefulSet(ctx context.Context, dk *dynakube.DynaKube, agCapability capability.Capability) (*appsv1.StatefulSet, error) {
	kubeUID := types.UID(dk.Status.KubeSystemUUID)

	activeGateConfigurationHash, err := r.calculateActiveGateConfigurationHash(ctx, dk, agCapability)
	if err != nil {
		return nil, err
	}

	statefulSetBuilder := NewStatefulSetBuilder(kubeUID, activeGateConfigurationHash, *dk, agCapability)

	desiredSts, err := statefulSetBuilder.CreateStatefulSet()
	if err != nil {
		return nil, err
	}

	if err = k8sstatefulset.ResolveAndSetReplicas(ctx, r.apiReader, desiredSts, dk.Spec.ActiveGate.Replicas); err != nil {
		return nil, err
	}

	return desiredSts, nil
}

func (r *Reconciler) calculateActiveGateConfigurationHash(ctx context.Context, dk *dynakube.DynaKube, agCapability capability.Capability) (string, error) {
	customPropertyData, err := r.getCustomPropertyValue(ctx, dk, agCapability)
	if err != nil {
		return "", err
	}

	authTokenData, err := r.getAuthTokenValue(ctx, dk)
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

func (r *Reconciler) getCustomPropertyValue(ctx context.Context, dk *dynakube.DynaKube, agCapability capability.Capability) (string, error) {
	if !needsCustomPropertyHash(agCapability.Properties().CustomProperties) {
		return "", nil
	}

	customPropertyData, err := r.getDataFromCustomProperty(ctx, dk, agCapability.Properties().CustomProperties)
	if err != nil {
		return "", err
	}

	return customPropertyData, nil
}

func (r *Reconciler) getAuthTokenValue(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	if !dk.ActiveGate().IsEnabled() {
		return "", nil
	}

	authTokenData, err := r.getDataFromAuthTokenSecret(ctx, dk)
	if err != nil {
		return "", err
	}

	return authTokenData, nil
}

func (r *Reconciler) getDataFromCustomProperty(ctx context.Context, dk *dynakube.DynaKube, customProperties *value.Source) (string, error) {
	if customProperties.ValueFrom != "" {
		return k8ssecret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{Namespace: dk.Namespace, Name: customProperties.ValueFrom}, customproperties.DataKey)
	}

	return customProperties.Value, nil
}

func (r *Reconciler) getDataFromAuthTokenSecret(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	return k8ssecret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{Namespace: dk.Namespace, Name: dk.ActiveGate().GetAuthTokenSecretName()}, authtoken.ActiveGateAuthTokenName)
}

func needsCustomPropertyHash(customProperties *value.Source) bool {
	return customProperties != nil && (customProperties.Value != "" || customProperties.ValueFrom != "")
}
