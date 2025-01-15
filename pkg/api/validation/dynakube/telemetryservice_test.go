package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
)

func TestTelemetryServiceProtocols(t *testing.T) {
	t.Run(`no list of protocols`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &telemetryservice.Spec{
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
					TelemetryService: &telemetryservice.Spec{
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
					TelemetryService: &telemetryservice.Spec{
						Protocols: []string{
							string(telemetryservice.ZipkinProtocol),
							string(telemetryservice.OtlpProtocol),
							"unknown",
						},
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
					TelemetryService: &telemetryservice.Spec{
						Protocols: []string{
							string(telemetryservice.ZipkinProtocol),
							string(telemetryservice.OtlpProtocol),
							"unknown1",
							"unknown2",
						},
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
					TelemetryService: &telemetryservice.Spec{
						Protocols: []string{
							string(telemetryservice.ZipkinProtocol),
							string(telemetryservice.OtlpProtocol),
							string(telemetryservice.OtlpProtocol),
						},
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
					TelemetryService: &telemetryservice.Spec{
						Protocols: []string{
							string(telemetryservice.ZipkinProtocol),
							string(telemetryservice.ZipkinProtocol),
							string(telemetryservice.OtlpProtocol),
							string(telemetryservice.OtlpProtocol),
							string(telemetryservice.JaegerProtocol),
						},
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
					TelemetryService: &telemetryservice.Spec{},
				},
			})
	})

	t.Run(`no telemetry service`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
				},
			})
	})

	t.Run(`service name too long`, func(t *testing.T) {
		assertDenied(t,
			[]string{invalidTelemetryServiceNameErrorMessage()},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &telemetryservice.Spec{
						ServiceName: "a123456789012345678901234567890123456789012345678901234567890123",
					},
				},
			})
	})

	t.Run(`service name violates DNS-1035`, func(t *testing.T) {
		assertDenied(t,
			[]string{invalidTelemetryServiceNameErrorMessage()},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &telemetryservice.Spec{
						ServiceName: "0123",
					},
				},
			})
	})
}
