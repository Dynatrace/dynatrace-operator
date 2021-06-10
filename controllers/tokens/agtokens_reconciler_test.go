package tokens

import (
	"os"
	"reflect"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

var instance *v1alpha1.DynaKube
var secretsData map[string][]byte
var moreSecretsData map[string][]byte
var lessSecretsData map[string][]byte

func init() {
	instance = &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: testName},
	}

	secretsData = map[string][]byte{
		"tokenname":    []byte("secretdata"),
		"anothertoken": []byte("topsecret"),
	}

	moreSecretsData = map[string][]byte{
		"tokenname":    []byte("secretdata"),
		"anothertoken": []byte("topsecret"),
		"extra":        []byte("aaa"),
	}

	lessSecretsData = map[string][]byte{
		"tokenname": []byte("secretdata"),
	}
}

func TestBuildAGTokensSecret(t *testing.T) {
	type args struct {
		instance           *v1alpha1.DynaKube
		agTokensSecretData map[string][]byte
	}

	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "",
			args: args{
				instance:           instance,
				agTokensSecretData: secretsData,
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretsName,
					Namespace: instance.Namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: secretsData,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildAGTokensSecret(tt.args.instance, tt.args.agTokensSecretData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildAGTokensSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createDefaultReconciler(fakeClient client.Client, dtc *dtclient.MockDynatraceClient) *Reconciler {
	return &Reconciler{
		Client:    fakeClient,
		apiReader: fakeClient,
		instance:  instance,
		dtc:       dtc,
		log:       zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
		scheme:    scheme.Scheme,
	}
}

func TestReconciler_getAGTokens(t *testing.T) {
	tenantToken := "blablabla"

	mockDtcClient := &dtclient.MockDynatraceClient{}
	fakeClient := fake.NewClient()

	mockDtcClient.On("GetAGTenantInfo").
		Return(&dtclient.TenantInfo{
			Token: tenantToken,
		}, nil)

	tests := []struct {
		name    string
		want    map[string][]byte
		wantErr bool
	}{
		{
			name:    "",
			want:    map[string][]byte{"tenant-token": []byte(tenantToken)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := createDefaultReconciler(fakeClient, mockDtcClient)
			got, err := r.getAGTokens()
			if (err != nil) != tt.wantErr {
				t.Errorf("getAGTokens() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAGTokens() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isAGTokensSecretEqual(t *testing.T) {
	type args struct {
		currentSecret *corev1.Secret
		desired       map[string][]byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal",
			args: args{
				currentSecret: BuildAGTokensSecret(&v1alpha1.DynaKube{}, secretsData),
				desired:       secretsData,
			},
			want: true,
		},
		{
			name: "inequal - subset of data",
			args: args{
				currentSecret: BuildAGTokensSecret(&v1alpha1.DynaKube{}, lessSecretsData),
				desired:       secretsData,
			},
			want: false,
		},
		{
			name: "inequal - superset of data",
			args: args{
				currentSecret: BuildAGTokensSecret(&v1alpha1.DynaKube{}, moreSecretsData),
				desired:       secretsData,
			},
			want: false,
		},
		{
			name: "inequal - completly different data",
			args: args{
				currentSecret: BuildAGTokensSecret(&v1alpha1.DynaKube{}, map[string][]byte{"other": []byte("data")}),
				desired:       secretsData,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAGTokensSecretEqual(tt.args.currentSecret, tt.args.desired); got != tt.want {
				t.Errorf("isAGTokensSecretEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
