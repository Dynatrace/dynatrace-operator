package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

func TestTelemetryServiceProtocols(t *testing.T) {
	t.Run(`no list of protocols`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{
						Protocols: nil,
					},
				},
			})
	})

	t.Run(`empty list of protocols`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{
						Protocols: []string{},
					},
				},
			})
	})

	t.Run(`unknown protocol`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryServiceUnknownProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{
						Protocols: []string{dynakube.TelemetryServiceZipkinProtocol, dynakube.TelemetryServiceOtlpProtocol, "unknown"},
					},
				},
			})
	})

	t.Run(`unknown protocols`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryServiceUnknownProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{
						Protocols: []string{dynakube.TelemetryServiceZipkinProtocol, dynakube.TelemetryServiceOtlpProtocol, "unknown1", "unknown2"},
					},
				},
			})
	})

	t.Run(`duplicated protocol`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryServiceDuplicatedProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{
						Protocols: []string{dynakube.TelemetryServiceZipkinProtocol, dynakube.TelemetryServiceOtlpProtocol, dynakube.TelemetryServiceOtlpProtocol},
					},
				},
			})
	})

	t.Run(`duplicated protocols`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryServiceDuplicatedProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{
						Protocols: []string{dynakube.TelemetryServiceZipkinProtocol, dynakube.TelemetryServiceZipkinProtocol, dynakube.TelemetryServiceOtlpProtocol, dynakube.TelemetryServiceOtlpProtocol, dynakube.TelemetryServiceJaegerProtocol},
					},
				},
			})
	})

	t.Run(`default config`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:           testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{},
				},
			})
	})
}
