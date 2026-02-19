package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testOtherName        = "test-other-name"
	testServiceName      = "test-service-name"
	testOtherServiceName = "test-other-service-name"
)

var otherDynakubeObjectMeta = metav1.ObjectMeta{
	Name:      testOtherName,
	Namespace: testNamespace,
}

func TestTelemetryIngestProtocols(t *testing.T) {
	t.Run("no list of protocols", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: nil,
					},
					Templates: dynakube.TemplatesSpec{
						OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
							ImageRef: image.Ref{
								Repository: "test-repo",
								Tag:        "test-tag",
							},
						},
					},
				},
			})
	})

	t.Run("empty list of protocols", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []otelcgen.Protocol{},
					},
					Templates: dynakube.TemplatesSpec{
						OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
							ImageRef: image.Ref{
								Repository: "test-repo",
								Tag:        "test-tag",
							},
						},
					},
				},
			})
	})

	t.Run("unknown protocol", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestUnknownProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []otelcgen.Protocol{
							otelcgen.ZipkinProtocol,
							otelcgen.OtlpProtocol,
							"unknown",
						},
					},
				},
			})
	})

	t.Run("unknown protocols", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestUnknownProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []otelcgen.Protocol{
							otelcgen.ZipkinProtocol,
							otelcgen.OtlpProtocol,
							"unknown1",
							"unknown2",
						},
					},
				},
			})
	})

	t.Run("duplicated protocol", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestDuplicatedProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []otelcgen.Protocol{
							otelcgen.ZipkinProtocol,
							otelcgen.OtlpProtocol,
							otelcgen.OtlpProtocol,
						},
					},
				},
			})
	})

	t.Run("duplicated protocols", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestDuplicatedProtocols},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						Protocols: []otelcgen.Protocol{
							otelcgen.ZipkinProtocol,
							otelcgen.ZipkinProtocol,
							otelcgen.OtlpProtocol,
							otelcgen.OtlpProtocol,
							otelcgen.JaegerProtocol,
						},
					},
				},
			})
	})

	t.Run("default config", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:          testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{},
					Templates: dynakube.TemplatesSpec{
						OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
							ImageRef: image.Ref{
								Repository: "test-repo",
								Tag:        "test-tag",
							},
						},
					},
				},
			})
	})

	t.Run("no telemetry service", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
				},
			})
	})

	t.Run("service name too long", func(t *testing.T) {
		assertDenied(t,
			[]string{invalidTelemetryIngestNameErrorMessage()},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "a123456789012345678901234567890123456789012345678901234567890123",
					},
				},
			})
	})

	t.Run("service name violates DNS-1035", func(t *testing.T) {
		assertDenied(t,
			[]string{invalidTelemetryIngestNameErrorMessage()},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "0123",
					},
				},
			})
	})
}

func TestConflictingServiceNames(t *testing.T) {
	t.Run("no conflicts", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:          testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{},
					Templates: dynakube.TemplatesSpec{
						OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
							ImageRef: image.Ref{
								Repository: "test-repo",
								Tag:        "test-tag",
							},
						},
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: otherDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:          testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{},
				},
			})
	})

	t.Run("custom service name vs custom service name", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestServiceNameInUse},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: testServiceName,
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: otherDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: testServiceName,
					},
				},
			})
	})

	t.Run("custom service name vs default service name", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestServiceNameInUse},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: testOtherName + "-telemetry-ingest",
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: otherDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:          testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{},
				},
			})
	})

	t.Run("default service name vs custom service name", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestServiceNameInUse},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:          testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: otherDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: testName + "-telemetry-ingest",
					},
				},
			})
	})
}

func TestForbiddenSuffix(t *testing.T) {
	t.Run("activegate", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestForbiddenServiceName},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "test" + "-" + agconsts.MultiActiveGateName,
					},
				},
			})
	})
	t.Run("extensions", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestForbiddenServiceName},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "test" + consts.ExtensionsControllerSuffix,
					},
				},
			})
	})
	t.Run("telemetry ingest", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestForbiddenServiceName},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "test" + telemetryingest.ServiceNameSuffix,
					},
				},
			})
	})
	t.Run("webhook", func(t *testing.T) {
		assertDenied(t,
			[]string{errorTelemetryIngestForbiddenServiceName},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{
						ServiceName: "test-webhook",
					},
				},
			})
	})
}

func TestImages(t *testing.T) {
	t.Run("otel collector image missing", func(t *testing.T) {
		assertDenied(t, []string{errorOtelCollectorMissingImage},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:          testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{},
				},
			})
	})

	t.Run("otel collector image present", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:          testAPIURL,
					TelemetryIngest: &telemetryingest.Spec{},
					Templates: dynakube.TemplatesSpec{
						OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{
							ImageRef: image.Ref{
								Repository: "test-repo",
								Tag:        "test-tag",
							},
						},
					},
				},
			})
	})
}
