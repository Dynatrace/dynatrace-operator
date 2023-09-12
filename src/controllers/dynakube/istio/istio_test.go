package istio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
)

func TestIstio(t *testing.T) {
	type test struct {
		name    string
		input   []*metav1.APIResourceList
		wantErr error
		want    bool
	}

	tests := []test{
		{name: "enabled", input: []*metav1.APIResourceList{{GroupVersion: IstioGVR}}, wantErr: nil, want: true},
		{name: "disabled", input: []*metav1.APIResourceList{}, wantErr: nil, want: false},
	}

	ist := fakeistio.NewSimpleClientset()

	fakeDiscovery, ok := ist.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeDiscovery.Resources = tc.input
			isInstalled, err := CheckIstioInstalled(fakeDiscovery)
			assert.Equal(t, tc.want, isInstalled)
			assert.ErrorIs(t, tc.wantErr, err)
		})
	}
}
