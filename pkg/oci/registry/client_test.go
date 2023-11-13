package registry

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_areCustomImagesAffectedByFeatureNoProxy(t *testing.T) {
	type args struct {
		dynakube *dynatracev1beta1.DynaKube
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "",
			args: args{
				dynakube: getClassicFullStackDynakube("", "", ""),
			},
			want: false,
		},
		{
			name: "",
			args: args{
				dynakube: getClassicFullStackDynakube("gcr.io", "", ""),
			},
			want: false,
		},
		{
			name: "",
			args: args{
				dynakube: getClassicFullStackDynakube("gcr.io", "gcr.io/superAgent", "gcr.io/superActiveGate"),
			},
			want: true,
		},
		{
			name: "",
			args: args{
				dynakube: getClassicFullStackDynakube("gcr.io", "gcr.io/superAgent", ""),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := areCustomImagesAffectedByFeatureNoProxy(tt.args.dynakube); got != tt.want {
				t.Errorf("areCustomImagesAffectedByFeatureNoProxy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getClassicFullStackDynakube(featureNoProxy string, OneAgentImage string, ActiveGateImage string) *dynatracev1beta1.DynaKube {
	dk := getDynakube(featureNoProxy, ActiveGateImage)
	dk.Spec = dynatracev1beta1.DynaKubeSpec{
		OneAgent: dynatracev1beta1.OneAgentSpec{
			ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
				Image: OneAgentImage,
			},
		},
	}
	return dk
}

func getCloudNativeFullStackDynakube(featureNoProxy string, OneAgentImage string, ActiveGateImage string) *dynatracev1beta1.DynaKube {
	dk := getDynakube(featureNoProxy, ActiveGateImage)
	dk.Spec = dynatracev1beta1.DynaKubeSpec{
		OneAgent: dynatracev1beta1.OneAgentSpec{
			CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
				HostInjectSpec: dynatracev1beta1.HostInjectSpec{
					Image: OneAgentImage,
				},
			},
		},
	}
	return dk
}

func getDynakube(featureNoProxy string, ActiveGateImage string) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"feature.dynatrace.com/no-proxy": featureNoProxy}},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				CapabilityProperties: dynatracev1beta1.CapabilityProperties{
					Image: ActiveGateImage,
				},
			},
		},
	}
}
