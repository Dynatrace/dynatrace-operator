package statefulset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder"
	agmodifiers "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/modifiers"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type statefulSetBuilder struct {
	kubeUID    types.UID
	configHash string
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func NewStatefulSetBuilder(kubeUID types.UID, configHash string, dynakube dynatracev1beta1.DynaKube, capability capability.Capability) statefulSetBuilder {
	return statefulSetBuilder{
		kubeUID:    kubeUID,
		configHash: configHash,
		dynakube:   dynakube,
		capability: capability,
	}
}

func (builder statefulSetBuilder) CreateStatefulSet(modifiers []agbuilder.Modifier) (*appsv1.StatefulSet, error) {
	activeGateBuilder := agbuilder.Builder{}
	baseStatefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        builder.dynakube.Name + "-" + builder.capability.ShortName(),
			Namespace:   builder.dynakube.Namespace,
			Annotations: map[string]string{},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            builder.capability.Properties().Replicas,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						consts.AnnotationActiveGateConfigurationHash: builder.configHash,
					},
				},
			},
		}}
	activeGateBuilder.SetBase(baseStatefulSet)
	if len(modifiers) == 0 {
		modifiers = agmodifiers.GetAllModifiers(builder.kubeUID, builder.dynakube, builder.capability)
	}
	for _, modifier := range modifiers {
		activeGateBuilder.AddModifier(modifier)
	}
	sts := activeGateBuilder.Build()

	hash, err := kubeobjects.GenerateHash(sts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	sts.ObjectMeta.Annotations[kubeobjects.AnnotationHash] = hash

	return &sts, nil
}
