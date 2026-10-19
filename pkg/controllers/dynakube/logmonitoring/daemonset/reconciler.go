// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdaemonset"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionType = "LogMonitoringDaemonSet"

	serviceAccountName = "dynatrace-logmonitoring"
)

type Reconciler struct {
	daemonset k8sdaemonset.QueryObject
}

func NewReconciler(clt client.Client,
	apiReader client.Reader) *Reconciler {
	return &Reconciler{
		daemonset: k8sdaemonset.Query(clt, apiReader),
	}
}

var KubernetesSettingsNotAvailableError = errors.New("the status of the DynaKube is missing information about the kubernetes monitored-entity, skipping LogMonitoring deployment until it is ready")

func (r *Reconciler) Reconcile(ctx context.Context, imageClient dtimage.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "daemonset")

	if !dk.LogMonitoring().IsStandalone() {
		if meta.FindStatusCondition(*dk.Conditions(), ConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		err := r.daemonset.Delete(ctx, &appsv1.DaemonSet{Name: dk.LogMonitoring().GetDaemonSetName(), Namespace: dk.Namespace})
		if err != nil {
			log.Error(err, "failed to clean-up LogMonitoring DaemonSet")
		}

		meta.RemoveStatusCondition(dk.Conditions(), ConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	if err := resolveImage(ctx, imageClient, dk); err != nil {
		return err
	}

	ds, err := r.generateDaemonSet(dk, dk.Status.LogMonitoring.ResolvedImage)
	if err != nil {
		return err
	}

	updated, err := r.daemonset.WithOwner(dk).CreateOrUpdate(ctx, ds)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ConditionType, err)

		return err
	}

	if updated {
		k8sconditions.SetDaemonSetOutdated(dk.Conditions(), ConditionType, dk.LogMonitoring().GetDaemonSetName()) // needed to reset the timestamp
		k8sconditions.SetDaemonSetCreated(dk.Conditions(), ConditionType, dk.LogMonitoring().GetDaemonSetName())
	}

	return nil
}

// resolveImage records the image to deploy in the status. The templates section takes
// precedence over the image reported by the public registry.
func resolveImage(ctx context.Context, imageClient dtimage.Client, dk *dynakube.DynaKube) error {
	if ref := dk.LogMonitoring().Template().ImageRef; ref.HasImage() {
		dk.Status.LogMonitoring.ResolvedImage = ref.String()

		return nil
	}

	imageURI, err := registry.ResolveImage(ctx, imageClient, dk.PublicRegistryOverride(), dtimage.LogModule)
	if err != nil {
		return err
	}

	dk.Status.LogMonitoring.ResolvedImage = imageURI

	return nil
}

func (r *Reconciler) generateDaemonSet(dk *dynakube.DynaKube, imageURI string) (*appsv1.DaemonSet, error) {
	tenantUUID, err := dk.TenantUUID()
	if err != nil {
		return nil, err
	}

	tag := ""
	if dk.Spec.Templates.LogMonitoring != nil {
		tag = dk.Spec.Templates.LogMonitoring.ImageRef.Tag
	}

	labels := k8slabel.New(k8slabel.LogMonitoring, dk.Name, tag)

	ds, err := k8sdaemonset.Build(dk, dk.LogMonitoring().GetDaemonSetName(), getContainer(dk, tenantUUID, imageURI),
		k8sdaemonset.SetInitContainer(getInitContainer(dk, tenantUUID, imageURI)),
		k8sdaemonset.SetAllLabels(labels.AsMap(), labels.AsSelector(), labels.AsSelector(), dk.LogMonitoring().Template().Labels),
		k8sdaemonset.SetAllAnnotations(nil, r.getAnnotations(dk)),
		k8sdaemonset.SetServiceAccount(serviceAccountName),
		k8sdaemonset.SetDNSPolicy(dk.LogMonitoring().Template().DNSPolicy),
		k8sdaemonset.SetAffinity(k8saffinity.NewMultiArchNodeAffinity()),
		k8sdaemonset.SetPriorityClass(dk.LogMonitoring().Template().PriorityClassName),
		k8sdaemonset.SetNodeSelector(dk.LogMonitoring().Template().NodeSelector),
		k8sdaemonset.SetTolerations(dk.LogMonitoring().Template().Tolerations),
		k8sdaemonset.SetPullSecret(dk.CustomPullSecretReferences()...),
		k8sdaemonset.SetUpdateStrategy(getUpdateStrategy(dk)),
		k8sdaemonset.SetVolumes(getVolumes(dk.Name)),
		k8sdaemonset.SetSecurityContext(buildPodSecurityContext()),
	)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func getUpdateStrategy(dk *dynakube.DynaKube) appsv1.DaemonSetUpdateStrategy {
	maxUnavailable := intstr.FromInt(dk.FF().GetOneAgentMaxUnavailable()) //nolint:staticcheck

	us := appsv1.DaemonSetUpdateStrategy{
		RollingUpdate: &appsv1.RollingUpdateDaemonSet{
			MaxUnavailable: &maxUnavailable,
		},
	}

	if dk.LogMonitoring().Template().RollingUpdate != nil {
		us.RollingUpdate = dk.LogMonitoring().Template().RollingUpdate
	}

	return us
}

func isMEConfigured(dk *dynakube.DynaKube) bool {
	return dk.Status.KubernetesClusterMEID != "" && dk.Status.KubernetesClusterName != ""
}

func buildPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		FSGroup: new(runAs),
	}
}
