package istio

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
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

func TestBuildNameForEndpoint(t *testing.T) {
	type args struct {
		name      string
		commHosts []dtclient.CommunicationHost
	}

	type test struct {
		name string
		args args
		want string
	}

	tests := []test{
		{
			name: "empty host list",
			args: args{name: "test", commHosts: []dtclient.CommunicationHost{}},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name: "single host in list",
			args: args{name: "test", commHosts: []dtclient.CommunicationHost{{Protocol: "http", Host: "mydynatrace.somedomain", Port: 27018}}},
			want: "f694c984dc5db78631ca837b529d044e46a8594607156d4a2a8b93e7d488e47c",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildNameForEndpoint(tc.args.name, tc.args.commHosts)
			assert.Equal(t, tc.want, result)
		})
	}
}
