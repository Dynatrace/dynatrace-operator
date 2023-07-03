package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/fake"
	client_testing "k8s.io/client-go/testing"
)

func createDiscoveryClient(enableOpenshiftGVR bool) func() (discovery.DiscoveryInterface, error) {
	return func() (discovery.DiscoveryInterface, error) {
		client := &fake.FakeDiscovery{
			Fake: &client_testing.Fake{},
		}

		if enableOpenshiftGVR {
			client.Fake.Resources = []*v1.APIResourceList{
				{
					GroupVersion: openshiftSecurityGVR,
				},
			}
		}

		return client, nil
	}
}

func TestPlatformResolver(t *testing.T) {
	t.Run("should detect openshift", func(t *testing.T) {
		platformResolver := Resolver{
			discoveryProvider: createDiscoveryClient(true),
		}

		assert.True(t, platformResolver.IsOpenshift(t))
	})
	t.Run("should detect kubernetes", func(t *testing.T) {
		platformResolver := Resolver{
			discoveryProvider: createDiscoveryClient(false),
		}

		assert.False(t, platformResolver.IsOpenshift(t))
	})
}
