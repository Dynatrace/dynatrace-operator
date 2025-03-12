package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
)

func TestTelemetryIngestProtocols(t *testing.T) {
	t.Run(`no list of protocols`, func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{
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
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []string{},
					},
				},
			})
	})

	t.Run(`unknown protocol`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestUnknownProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []string{
							string(otelcgen.ZipkinProtocol),
							string(otelcgen.OtlpProtocol),
							"unknown",
						},
					},
				},
			})
	})

	t.Run(`unknown protocols`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestUnknownProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []string{
							string(otelcgen.ZipkinProtocol),
							string(otelcgen.OtlpProtocol),
							"unknown1",
							"unknown2",
						},
					},
				},
			})
	})

	t.Run(`duplicated protocol`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestDuplicatedProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []string{
							string(otelcgen.ZipkinProtocol),
							string(otelcgen.OtlpProtocol),
							string(otelcgen.OtlpProtocol),
						},
					},
				},
			})
	})

	t.Run(`duplicated protocols`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestDuplicatedProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []string{
							string(otelcgen.ZipkinProtocol),
							string(otelcgen.ZipkinProtocol),
							string(otelcgen.OtlpProtocol),
							string(otelcgen.OtlpProtocol),
							string(otelcgen.JaegerProtocol),
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
					APIURL:          testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{},
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
			[]string{invalidTelemetryIngestNameErrorMessage()},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "a123456789012345678901234567890123456789012345678901234567890123",
					},
				},
			})
	})

	t.Run(`service name violates DNS-1035`, func(t *testing.T) {
		assertDenied(t,
			[]string{invalidTelemetryIngestNameErrorMessage()},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "0123",
					},
				},
			})
	})
}
