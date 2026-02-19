//go:build e2e

package k8snamespace

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	InjectionKey          = "istio-injection"
	InjectionEnabledValue = "enabled"
)

type Option func(namespace *corev1.Namespace)

func New(name string, opts ...Option) *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, opt := range opts {
		opt(namespace)
	}

	return namespace
}

func WithLabels(labels map[string]string) Option {
	return func(namespace *corev1.Namespace) {
		namespace.Labels = labels
	}
}

func WithIstio() Option {
	return func(namespace *corev1.Namespace) {
		if namespace.Labels == nil {
			namespace.Labels = map[string]string{}
		}
		namespace.Labels[InjectionKey] = InjectionEnabledValue
	}
}

func WithAnnotation(annotations map[string]string) Option {
	return func(namespace *corev1.Namespace) {
		namespace.Annotations = annotations
	}
}

func Delete(namespaceName string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		err := envConfig.Client().Resources().Delete(ctx, &namespace, func(options *metav1.DeleteOptions) {
			options.GracePeriodSeconds = ptr.To[int64](0)
		})

		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}
			require.NoError(t, err)

			return ctx
		}

		resources := envConfig.Client().Resources()
		err = wait.For(conditions.New(resources).ResourceDeleted(&namespace), wait.WithTimeout(1*time.Minute))
		require.NoError(t, err)

		return ctx
	}
}

func Create(namespace corev1.Namespace) features.Func {
	return helpers.ToFeatureFunc(CreateForEnv(namespace), true)
}

func CreateForEnv(namespace corev1.Namespace) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		err := envConfig.Client().Resources().Create(ctx, &namespace)
		if k8serrors.IsAlreadyExists(err) {
			err = envConfig.Client().Resources().Update(ctx, &namespace)
		}
		if err != nil {
			return ctx, err
		}

		return AddIstioNetworkAttachment(namespace)(ctx, envConfig)
	}
}

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
