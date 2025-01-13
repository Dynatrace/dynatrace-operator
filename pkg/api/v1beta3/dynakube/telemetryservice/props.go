package telemetryservice

type Protocol string

const (
	OtlpProtocol   Protocol = "otlp"
	ZipkinProtocol Protocol = "zipkin"
	JaegerProtocol Protocol = "jaeger"
	StatsdProtocol Protocol = "statsd"
)

func KnownProtocols() []Protocol {
	return []Protocol{
		OtlpProtocol,
		ZipkinProtocol,
		JaegerProtocol,
		StatsdProtocol,
	}
}
