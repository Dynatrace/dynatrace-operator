package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestIsIstioNotInstalled(t *testing.T) {
	noIstioInterceptor := interceptor.Funcs{
		Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
			return new(meta.NoResourceMatchError)
		},
	}

	t.Run("istio is not installed", func(t *testing.T) {
		assertDeniedWithInterceptor(t, noIstioInterceptor, []string{errorNoIstioInstalled}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:      testAPIURL,
				EnableIstio: true,
			},
		})
	})

	t.Run("istio resources", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:      testAPIURL,
				EnableIstio: true,
			},
		})
	})

	t.Run("no istio resources + no istio enable -> no problem", func(t *testing.T) {
		assertAllowedWithInterceptor(t, noIstioInterceptor, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:      testAPIURL,
				EnableIstio: false,
			},
		})
	})
}
