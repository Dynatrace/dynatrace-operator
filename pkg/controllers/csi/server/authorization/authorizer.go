package authorization

import (
	"context"
	"errors"

	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/app"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/host"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errAccessDenied = errors.New("access denied")

// Authorizer verifies that a NodePublishVolume request is permitted to
// mount the requested volume
type Authorizer struct {
	apiReader         client.Reader
	operatorNamespace string
}

func New(apiReader client.Reader, operatorNamespace string) *Authorizer {
	return &Authorizer{
		apiReader:         apiReader,
		operatorNamespace: operatorNamespace,
	}
}

// Authorize checks whether the CSI request is allowed for the given mode and
// returns the DynaKube name (derived from the API server) on success
func (a *Authorizer) Authorize(ctx context.Context, cfg csivolumes.VolumeConfig) (string, error) {
	switch cfg.Mode {
	case appvolumes.Mode:
		return a.authorizeApp(ctx, cfg)
	case hostvolumes.Mode:
		return a.authorizeHost(ctx, cfg)
	default:
		log := logd.FromContext(ctx)
		log.Info("access denied: unknown csi mode", "mode", cfg.Mode)

		return "", errAccessDenied
	}
}

func (a *Authorizer) authorizeApp(ctx context.Context, cfg csivolumes.VolumeConfig) (string, error) {
	log := logd.FromContext(ctx)

	var ns corev1.Namespace
	if err := a.apiReader.Get(ctx, client.ObjectKey{Name: cfg.PodNamespace}, &ns); err != nil {
		log.Info("access denied: failed to get namespace", "namespace", cfg.PodNamespace, "error", err.Error())

		return "", errAccessDenied
	}

	dkName, ok := ns.Labels[dtwebhook.InjectionInstanceLabel]
	if !ok || dkName == "" {
		log.Info("access denied: namespace has no injection instance label", "namespace", cfg.PodNamespace)

		return "", errAccessDenied
	}

	if dkName != cfg.DynakubeName {
		log.Info("access denied: dynakube attribute mismatch", "namespace", cfg.PodNamespace, "expected", dkName, "got", cfg.DynakubeName)

		return "", errAccessDenied
	}

	return dkName, nil
}

func (a *Authorizer) authorizeHost(ctx context.Context, cfg csivolumes.VolumeConfig) (string, error) {
	log := logd.FromContext(ctx)

	if cfg.PodNamespace != a.operatorNamespace {
		log.Info("access denied: host mode request outside operator namespace", "namespace", cfg.PodNamespace)

		return "", errAccessDenied
	}

	var pod corev1.Pod
	if err := a.apiReader.Get(ctx, client.ObjectKey{Name: cfg.PodName, Namespace: cfg.PodNamespace}, &pod); err != nil {
		log.Info("access denied: failed to get pod", "pod", cfg.PodName, "namespace", cfg.PodNamespace, "error", err.Error())

		return "", errAccessDenied
	}

	if string(pod.UID) != cfg.PodUID {
		log.Info("access denied: pod UID mismatch", "pod", cfg.PodName, "namespace", cfg.PodNamespace)

		return "", errAccessDenied
	}

	if pod.Labels[k8slabel.AppManagedByLabel] != version.AppName {
		log.Info("access denied: pod missing managed-by label", "pod", cfg.PodName, "namespace", cfg.PodNamespace)

		return "", errAccessDenied
	}

	if pod.Labels[k8slabel.AppNameLabel] != k8slabel.OneAgentComponentLabel {
		log.Info("access denied: pod missing name=oneagent label", "pod", cfg.PodName, "namespace", cfg.PodNamespace)

		return "", errAccessDenied
	}

	dkName := pod.Labels[k8slabel.AppCreatedByLabel]
	if dkName == "" {
		log.Info("access denied: pod missing created-by label", "pod", cfg.PodName, "namespace", cfg.PodNamespace)

		return "", errAccessDenied
	}

	if dkName != cfg.DynakubeName {
		log.Info("access denied: dynakube attribute mismatch", "pod", cfg.PodName, "namespace", cfg.PodNamespace, "expected", dkName, "got", cfg.DynakubeName)

		return "", errAccessDenied
	}

	return dkName, nil
}
