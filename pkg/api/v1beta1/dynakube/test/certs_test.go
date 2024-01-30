package test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"software.sslmate.com/src/go-pkcs12"
)

const (
	testConfigMapName  = "test-config-map"
	testConfigMapValue = "test-config-map-value"

	testSecretName  = "test-secret"
	testSecretValue = "test-secret-value"

	testPassword = "test"
)

func TestCerts(t *testing.T) {
	t.Run(`get trusted certificate authorities`, trustedCAsTester)
	t.Run(`get no tls certificates`, activeGateTlsNoCertificateTester)
	activeGateTlsCertificate(t)
	t.Run(`get tls certificates extracted from p12`, activeGateTlsP12OnlyTester)
}

func trustedCAsTester(t *testing.T) {
	kubeReader := fake.NewClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: testConfigMapName},
		Data: map[string]string{
			dynakube.TrustedCAKey: testConfigMapValue,
		},
	})
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			TrustedCAs: testConfigMapName,
		},
	}
	trustedCAs, err := dk.TrustedCAs(context.TODO(), kubeReader)
	require.NoError(t, err)
	assert.Equal(t, []byte(testConfigMapValue), trustedCAs)

	kubeReader = fake.NewClient()
	trustedCAs, err = dk.TrustedCAs(context.TODO(), kubeReader)

	require.Error(t, err)
	assert.Empty(t, trustedCAs)

	emptyDk := dynakube.DynaKube{}
	trustedCAs, err = emptyDk.TrustedCAs(context.TODO(), kubeReader)
	require.NoError(t, err)
	assert.Empty(t, trustedCAs)
}

func activeGateTlsNoCertificateTester(t *testing.T) {
	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: dynakube.ActiveGateSpec{
				Capabilities:  []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName},
				TlsSecretName: testSecretName,
			},
		},
	}

	kubeReader := fake.NewClient()
	tlsCert, err := dk.ActiveGateTlsCert(context.TODO(), kubeReader)

	require.Error(t, err)
	assert.Empty(t, tlsCert)

	emptyDk := dynakube.DynaKube{}
	tlsCert, err = emptyDk.ActiveGateTlsCert(context.TODO(), kubeReader)

	require.NoError(t, err)
	assert.Empty(t, tlsCert)
}

func activeGateTlsCertificate(t *testing.T) {
	testFunc := func(t *testing.T, data map[string][]byte) {
		kubeReader := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
			Data:       data,
		})

		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: dynakube.ActiveGateSpec{
					Capabilities:  []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName},
					TlsSecretName: testSecretName,
				},
			},
		}
		tlsCert, err := dk.ActiveGateTlsCert(context.TODO(), kubeReader)

		require.NoError(t, err)
		assert.Equal(t, testSecretValue, tlsCert)
	}

	t.Run("get tls certificates from server.crt", func(t *testing.T) {
		testFunc(t, map[string][]byte{
			dynakube.TlsCertKey: []byte(testSecretValue),
		})
	})

	t.Run("get tls certificates from server.crt if .crt and .p12 provided", func(t *testing.T) {
		testFunc(t, map[string][]byte{
			dynakube.TlsCertKey:        []byte(testSecretValue),
			dynakube.TlsP12Key:         []byte("bla"),
			dynakube.TlsP12PasswordKey: []byte(testPassword),
		})
	})
}

func activeGateTlsP12OnlyTester(t *testing.T) {
	// create binary p12 container instead of loading a .p12 file
	testAgCert := `-----BEGIN CERTIFICATE-----
MIIDqDCCApCgAwIBAgIULcjRQPD6pITUZk9/40KxsNNC7F8wDQYJKoZIhvcNAQEL
BQAwJTEjMCEGA1UEAwwaZHluYWt1YmUtYWN0aXZlZ2F0ZS5pc3N1ZXIwHhcNMjQw
MzI2MTMyNDI3WhcNMjUwMzI2MTMyNDI3WjAoMSYwJAYDVQQDDB1keW5ha3ViZS1h
Y3RpdmVnYXRlLmR5bmF0cmFjZTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAKLFbBNd34ew22xZoaHr6rcxl1sZ98d0lkxZIu+rjOfBH51CwKN8M/nqXoRm
rbn6qCEP3PtgHneSjorS8OKh9kzl3otnQPh7IAl+p8SxaZ5NzXwrT6/pL6cUBMgR
DeywNTR9wwfQNSPZm9sEvveOE5hGQpGJQkYF9K8Z3YAqRUrKVuEyu7VzBy5ND1nB
gK5/JZ6hV5D1SXtYsupLBp2j/BWB6CY5cD9H/4ThUvoHhONNKcjWhs5CRBWYLfiz
9B8h3ov0PSgEU/5ze+TVUiQBLegG5ikSFNLLqldT7yGt3No4CWPkmoOiGY8s+HuH
lyeDibbwEjU2gWfo2G3S7K5oRTECAwEAAaOBzDCByTAfBgNVHSMEGDAWgBTo8pkG
JpkcUCHwtVTZvpT2TBeaZDAJBgNVHRMEAjAAMHwGA1UdEQR1MHOCHWR5bmFrdWJl
LWFjdGl2ZWdhdGUuZHluYXRyYWNlgiFkeW5ha3ViZS1hY3RpdmVnYXRlLmR5bmF0
cmFjZS5zdmOCL2R5bmFrdWJlLWFjdGl2ZWdhdGUuZHluYXRyYWNlLnN2Yy5jbHVz
dGVyLmxvY2FsMB0GA1UdDgQWBBQKq4Bbt7ulXa5a3Ya/YWFXLW1jwjANBgkqhkiG
9w0BAQsFAAOCAQEAw8UhxX4twpNaFPkr/KwNhkciQ91vZsk+TTu/poQEaS3pkywD
RJ23x54d/Y2mDxWEdMeOaPr5YX8DmZ6UQHzhcdhqQPJ6j/hNG4McpRyp3EnzRhcl
2fct+YmiF/AUL2VAHInec4OAIu0XuE6kGIbGVPShAFbcTF4ee+liyFVPH/4cutqF
sBYbkfs8u5BLG7Tk/Gp8hpSYqy26Ior7wXvw1ZhoRitbUYJSlM61aHKAs8NbHSaU
mkMu4GN9N03PCAMqrQ9SG0tQ3yaD0uMdFn4bhs238nMVRS9/rFfuGrnGjeu/Dvtu
fLq5D5lkcuMC7MPZLExDdr+js10zWZg918GkVQ==
-----END CERTIFICATE-----
`
	testAgKey := `-----BEGIN PRIVATE KEY-----
MIIEugIBADANBgkqhkiG9w0BAQEFAASCBKQwggSgAgEAAoIBAQCixWwTXd+HsNts
WaGh6+q3MZdbGffHdJZMWSLvq4znwR+dQsCjfDP56l6EZq25+qghD9z7YB53ko6K
0vDiofZM5d6LZ0D4eyAJfqfEsWmeTc18K0+v6S+nFATIEQ3ssDU0fcMH0DUj2Zvb
BL73jhOYRkKRiUJGBfSvGd2AKkVKylbhMru1cwcuTQ9ZwYCufyWeoVeQ9Ul7WLLq
Swado/wVgegmOXA/R/+E4VL6B4TjTSnI1obOQkQVmC34s/QfId6L9D0oBFP+c3vk
1VIkAS3oBuYpEhTSy6pXU+8hrdzaOAlj5JqDohmPLPh7h5cng4m28BI1NoFn6Nht
0uyuaEUxAgMBAAECgf9zrtwvIF3oWWZEG2/S7AOBoVmp5byX2ouiDNRHns+wyknY
bKcLGGWfgDnJGL7uC2I55FNxOHuLRx3rNwBSEjvQFgEi9hTTaHWjnzXyo3ntM0Ie
tqmIarKBqONLTbc0Oc7r4yBye10QSFFSEMZTR1+ly5717Pse5arzAJmzUpB6+aFA
SwMAarN47PSxD+5z3ZUTWtOWae+CZ0B2GmedYD+2QwTRlDWH4mVJMdT8JUosherq
MejqqSMTlZT3tMhC5TRQI8y7o77shZs/yTNfjxiXxbyS4WPM+Or4ockhJVFgKTVj
hdId1GXX2AfTNsn+TS0E+Rt1U5SBqt0KUloaok8CgYEA5PqtFYr5a5NwSi/1SD3w
SN2VmIMIesoRDh+wrHnJwqLBWei4GzpfxGsdSiHNtUmzUfnhf8NEY1pNkv9ETGmt
1dtugd7vNvehmhZn8CinIMv9/08rrzUITrF3cj9MmAZv1+0MdKaGxHdTI0f4fUrh
yix+MDRYtM4vZoSPybUQ7usCgYEAtfqkO9VzYcKgwpM4i1rg47c1ABfTrtRylM0S
603TEYl76YFfrkn0S0a92YjqtZbk1WT1lj7zy/78cvUfVnW7Iw0225dp9JpCPq3n
jTgdH5P5HgUKU8YOo5BLhAVC24FENJlhvchWQDazKTqQCOuXRSW2Gf1xK38wmXwJ
IZfirVMCgYA2rtLU+Tp1gWFopilalkgi7qACKxDEWitWhyTnG7KeQ8YPFa+Z+QfT
3YzCHm6E49PqONWscFKNs4whFcsWwIoeL2glpbrVErBKHx21UdAP2gePiDWzguO3
/1O6OfmtuKPPGjJGTVqT4rc9Drv+F/ryEEwWcPnaO/8/6Vp5Xj9r/wKBgEVqaVFl
l5C39CikjdIihVx3myEA9b9fzKFUJJ5bXmL3JawprHzIOwan4m1jW9yOxZVc4I1C
UC8Fgfi75gtN92dkeAOFm2YxnYlZPtVQjVNpV4KK+6h/CUB9H0Ep3Jnskj7aLz18
eOIfu3HDpAOzEk3PF8qMMaoc50X02WrWDCJ/AoGAcX9bQW8zg2TX6X0sUEpYUAIl
yEQmboVKXB9eXF4L+jrcoRYv7eI2TEo7vE8sx3e/DuDGXar6yDcwoekWqTCrUiFJ
W6ZPJ10fipXrHEZMW1q8riRh6mk0h3YGn6j0Ur4dCTTLQ7b58MPms9lw0qnDALxf
l0n5kcxfsqzO7Qww7PE=
-----END PRIVATE KEY-----
`
	testRootCert := `-----BEGIN CERTIFICATE-----
MIIDKzCCAhOgAwIBAgIUcVd72sFBwSgZGDarsNYeQj0impswDQYJKoZIhvcNAQEL
BQAwJTEjMCEGA1UEAwwaZHluYWt1YmUtYWN0aXZlZ2F0ZS5pc3N1ZXIwHhcNMjQw
MzI2MTMyNDI3WhcNMjkwMzI1MTMyNDI3WjAlMSMwIQYDVQQDDBpkeW5ha3ViZS1h
Y3RpdmVnYXRlLmlzc3VlcjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AMgvAQC6+gdU+C5UtjiGVc/EEac7GjFr2kTsuW0clDh3J2PKQ+8f3DxwxHVR7u/i
mzNdvG6QnugTug1TrxK3GhYzAvvZI3nHjRYjl+HwUxNn5BgodncYil1vNLXzIDI5
LI0xXGwL/YtZ1rE3mg302LiT4l8MUg6xPClKLzrt7zVskbzjsm080OaRAVmvqlhB
3pN1N34ephk/bby1E5BMz0BRivvrgkjuey6CTEtSfRubavmuz0af+6UT26ur3x76
qZpoRh1db0yBabAgDzihN7+s4mPBGQQ+IeQEYUe/go6nQS2QbITPFgDVgLFSWoVn
U+O1ilS40QYcgj565sXKS3UCAwEAAaNTMFEwHQYDVR0OBBYEFOjymQYmmRxQIfC1
VNm+lPZMF5pkMB8GA1UdIwQYMBaAFOjymQYmmRxQIfC1VNm+lPZMF5pkMA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBABgre1gSG9yNY3pwNHwmtv8a
4pMb8mmk2IZ+qQBkZ1D8YhCDRALWfB6YFdfaGVzX248EA00rZnzsUVMp5nOEejUt
+4KsyCoPy+tTnp+496g/7s9VuQRLKQkCoE5GwsuT/FWM81jb4Vnpz/eAonqx31Fl
Tfe6I8lORDlcqcg3NZ6HeVCrUuG8uQc0UHu62ubCWBP7VW+M/hUUOiGk3kg2U75t
7QfqjXNU7d6xykvlxYzAJC33gyrXtR3TSmw9GPGR1fVHA6SgizhDFjXw+uw1eP8B
X+Zt67UMVTP1HhJ1YYs/xRDwoDm5izZYjAirP9Hrnhw7CFgvEwBDh39uB2vJ0dw=
-----END CERTIFICATE-----
`

	testAgCertBlock, rest := pem.Decode([]byte(testAgCert))
	require.Empty(t, rest)

	testAgX509Cert, err := x509.ParseCertificate(testAgCertBlock.Bytes)
	require.NoError(t, err)

	testAgKeyBlock, rest := pem.Decode([]byte(testAgKey))
	require.Empty(t, rest)

	testAgX509Key, err := x509.ParsePKCS8PrivateKey(testAgKeyBlock.Bytes)
	require.NoError(t, err)

	testRootCertBlock, rest := pem.Decode([]byte(testRootCert))
	require.Empty(t, rest)

	testRootX509Cert, err := x509.ParseCertificate(testRootCertBlock.Bytes)
	require.NoError(t, err)

	p12Data, err := pkcs12.Modern.Encode(testAgX509Key, testAgX509Cert, []*x509.Certificate{testRootX509Cert}, testPassword)
	require.NoError(t, err)

	kubeReader := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
		Data: map[string][]byte{
			dynakube.TlsP12Key:         p12Data,
			dynakube.TlsP12PasswordKey: []byte(testPassword),
		}})

	dk := dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: dynakube.ActiveGateSpec{
				Capabilities:  []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName},
				TlsSecretName: testSecretName,
			},
		},
	}
	tlsCert, err := dk.ActiveGateTlsCert(context.TODO(), kubeReader)
	require.NoError(t, err)
	assert.Equal(t, testAgCert+"\n"+testRootCert+"\n", tlsCert)
}
