package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/fake"
	client_testing "k8s.io/client-go/testing"
)

type platformResolverTest struct {
	enableOpenshiftGVR bool
}

func (p *platformResolverTest) getDiscoveryClient() (discovery.DiscoveryInterface, error) {
	client := &fake.FakeDiscovery{
		Fake: &client_testing.Fake{},
	}

	if p.enableOpenshiftGVR {
		client.Fake.Resources = []*v1.APIResourceList{
			{
				GroupVersion: SccGVR,
			},
		}
	}

	return client, nil
}

func TestPlatformResolver(t *testing.T) {
	t.Run("should detect openshift", func(t *testing.T) {
		platformResolver := PlatformResolver{
			discoveryClientCreation: &platformResolverTest{
				enableOpenshiftGVR: true,
			},
		}

		assert.True(t, platformResolver.IsOpenshift(t))
	})
	t.Run("should detect kubernetes", func(t *testing.T) {
		platformResolver := PlatformResolver{
			discoveryClientCreation: &platformResolverTest{
				enableOpenshiftGVR: false,
			},
		}

		assert.False(t, platformResolver.IsOpenshift(t))
	})
}
