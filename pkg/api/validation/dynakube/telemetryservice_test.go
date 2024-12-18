package validation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"testing"
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
		assertDenied(t,
			[]string{errorTelemetryServiceNotEnoughProtocols},
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

	t.Run(`too many protocols`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryServiceTooManyProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					TelemetryService: &dynakube.TelemetryServiceSpec{
						Protocols: []string{"a", "b", "c", "d", "e"},
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
						Protocols: []string{"zipkin", "otlp", "unknown"},
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
						Protocols: []string{"zipkin", "otlp", "unknown1", "unknown2"},
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
