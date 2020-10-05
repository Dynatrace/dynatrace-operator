package builder

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	logger := logf.Log.WithName("builder")
	// Register OneAgent and Istio object schemas.
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		logger.Error(err, err.Error())
	}
	if err := os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace); err != nil {
		logger.Error(err, err.Error())
	}
}

func TestBuildDynatraceClient(t *testing.T) {
	secrets := make(map[string][]byte)
	secrets[_const.DynatraceApiToken] = []byte("some-api-token")
	secrets[_const.DynatracePaasToken] = []byte("some-paas-token")
	secrets[Proxy] = []byte("proxy-url")

	configMap := make(map[string]string)
	configMap["certs"] = "a certificate"

	proxyName := "test-proxy"
	configMapName := "test-map"

	t.Run("BuildDynatraceClient full", func(t *testing.T) {
		rtc := fake.NewFakeClientWithScheme(scheme.Scheme,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: proxyName,
				},
				Data: secrets,
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
				},
				Data: configMap,
			},
		)
		instance := dynatracev1alpha1.ActiveGate{
			Spec: dynatracev1alpha1.ActiveGateSpec{
				BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
					APIURL: "some-url",
					Proxy: &dynatracev1alpha1.ActiveGateProxy{
						ValueFrom: proxyName,
					},
					SkipCertCheck: true,
					TrustedCAs:    configMapName,
				},
			},
		}

		secret := corev1.Secret{
			Data: secrets,
		}
		dtc, err := BuildDynatraceClient(rtc, &instance, &secret)
		assert.NoError(t, err)
		assert.NotNil(t, dtc)
	})
	t.Run("BuildDynatraceClient minimal", func(t *testing.T) {
		instance := dynatracev1alpha1.ActiveGate{
			Spec: dynatracev1alpha1.ActiveGateSpec{
				BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
					APIURL: "some-url",
				},
			},
		}
		secret := corev1.Secret{
			Data: secrets,
		}
		dtc, err := BuildDynatraceClient(nil, &instance, &secret)
		assert.NoError(t, err)
		assert.NotNil(t, dtc)
	})
	t.Run("BuildDynatraceClient proxy config", func(t *testing.T) {
		t.Run("proxy secret not found", func(t *testing.T) {
			rtc := fake.NewFakeClientWithScheme(scheme.Scheme)
			instance := dynatracev1alpha1.ActiveGate{
				Spec: dynatracev1alpha1.ActiveGateSpec{
					BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
						APIURL: "some-url",
						Proxy: &dynatracev1alpha1.ActiveGateProxy{
							ValueFrom: proxyName,
						},
						SkipCertCheck: true,
						TrustedCAs:    configMapName,
					},
				},
			}

			secret := corev1.Secret{
				Data: secrets,
			}
			dtc, err := BuildDynatraceClient(rtc, &instance, &secret)
			assert.Error(t, err)
			assert.Nil(t, dtc)
			assert.Equal(t,
				"failed to get proxy secret: secrets \""+proxyName+"\" not found",
				err.Error())
		})
		t.Run("proxy secret misconfigured", func(t *testing.T) {
			rtc := fake.NewFakeClientWithScheme(scheme.Scheme,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: proxyName,
					},
				})
			instance := dynatracev1alpha1.ActiveGate{
				Spec: dynatracev1alpha1.ActiveGateSpec{
					BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
						APIURL: "some-url",
						Proxy: &dynatracev1alpha1.ActiveGateProxy{
							ValueFrom: proxyName,
						},
						SkipCertCheck: true,
						TrustedCAs:    configMapName,
					},
				},
			}

			secret := corev1.Secret{
				Data: secrets,
			}
			dtc, err := BuildDynatraceClient(rtc, &instance, &secret)
			assert.Error(t, err)
			assert.Nil(t, dtc)
			assert.Equal(t,
				"failed to extract proxy secret field: missing token proxy",
				err.Error())
		})
		t.Run("proxy secret from value", func(t *testing.T) {
			rtc := fake.NewFakeClientWithScheme(scheme.Scheme,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: proxyName,
					},
				})
			instance := dynatracev1alpha1.ActiveGate{
				Spec: dynatracev1alpha1.ActiveGateSpec{
					BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
						APIURL: "some-url",
						Proxy: &dynatracev1alpha1.ActiveGateProxy{
							Value: string(secrets[Proxy]),
						},
						SkipCertCheck: true,
					},
				},
			}

			secret := corev1.Secret{
				Data: secrets,
			}
			dtc, err := BuildDynatraceClient(rtc, &instance, &secret)
			assert.NoError(t, err)
			assert.NotNil(t, dtc)
		})
	})
	t.Run("BuildDynatraceClient certificate config errors", func(t *testing.T) {
		t.Run("certificate config missing", func(t *testing.T) {
			rtc := fake.NewFakeClientWithScheme(scheme.Scheme)
			instance := dynatracev1alpha1.ActiveGate{
				Spec: dynatracev1alpha1.ActiveGateSpec{
					BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
						APIURL:        "some-url",
						SkipCertCheck: true,
						TrustedCAs:    configMapName,
					},
				},
			}

			secret := corev1.Secret{
				Data: secrets,
			}
			dtc, err := BuildDynatraceClient(rtc, &instance, &secret)
			assert.Error(t, err)
			assert.Nil(t, dtc)
			assert.Equal(t,
				"failed to get certificate configmap: configmaps \""+configMapName+"\" not found",
				err.Error())
		})
		t.Run("certificate config misconfigured", func(t *testing.T) {
			rtc := fake.NewFakeClientWithScheme(scheme.Scheme,
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: configMapName,
					},
					//Data: configMap,
				})
			instance := dynatracev1alpha1.ActiveGate{
				Spec: dynatracev1alpha1.ActiveGateSpec{
					BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
						APIURL:        "some-url",
						SkipCertCheck: true,
						TrustedCAs:    configMapName,
					},
				},
			}

			secret := corev1.Secret{
				Data: secrets,
			}
			dtc, err := BuildDynatraceClient(rtc, &instance, &secret)
			assert.Error(t, err)
			assert.Nil(t, dtc)
			assert.Equal(t,
				"failed to extract certificate configmap field: missing field certs",
				err.Error())
		})
	})
	t.Run("BuildDynatraceClient all values nil", func(t *testing.T) {
		dtc, err := BuildDynatraceClient(nil, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
	t.Run("BuildDynatraceClient instance only", func(t *testing.T) {
		instance := dynatracev1alpha1.ActiveGate{}
		dtc, err := BuildDynatraceClient(nil, &instance, nil)
		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
	t.Run("BuildDynatraceClient instance and secret", func(t *testing.T) {
		instance := dynatracev1alpha1.ActiveGate{}
		secret := corev1.Secret{
			Data: secrets,
		}
		dtc, err := BuildDynatraceClient(nil, &instance, &secret)
		assert.Error(t, err)
		assert.Nil(t, dtc)
	})
}
