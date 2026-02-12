//go:build e2e

package k8snamespace

import (
	"context"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	InjectionKey          = "istio-injection"
	InjectionEnabledValue = "enabled"
)

var networkAttachmentPath = filepath.Join(project.TestDataDir(), "network/ocp-istio-cni.yaml")

func AddIstioNetworkAttachment(namespace corev1.Namespace) func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		isOpenshift, err := platform.NewResolver().IsOpenshift()
		if err != nil {
			return ctx, err
		}
		if !isOpenshift {
			return ctx, nil
		}
		if namespace.Labels[InjectionKey] == InjectionEnabledValue {
			ctx, err = manifests.InstallFromFile(networkAttachmentPath, decoder.MutateNamespace(namespace.Name))(ctx, envConfig)
			if err != nil {
				return ctx, err
			}
		}

		return ctx, nil
	}
}
