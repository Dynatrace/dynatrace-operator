package otelcgen

import (
	"github.com/pkg/errors"
	"go.opentelemetry.io/collector/component"
)

func (c *Config) buildReceiverComponent(componentID component.ID) component.Config {
	switch componentID {
	case OtlpID:
		return map[string]any{"protocols": map[string]any{
			"grpc": &ServerConfig{TLSSetting: c.buildTLSSetting(), Endpoint: c.buildEndpoint(OtlpGrpcPort)},
			"http": &ServerConfig{TLSSetting: c.buildTLSSetting(), Endpoint: c.buildEndpoint(OtlpHTTPPort)},
		}}
	case JaegerID:
		return map[string]any{"protocols": map[string]any{
			"grpc":           &ServerConfig{Endpoint: c.buildEndpoint(JaegerGrpcPort), TLSSetting: c.buildTLSSetting()},
			"thrift_binary":  &ServerConfig{Endpoint: c.buildEndpoint(JaegerThriftBinaryPort)},
			"thrift_compact": &ServerConfig{Endpoint: c.buildEndpoint(JaegerThriftCompactPort)},
			"thrift_http":    &ServerConfig{Endpoint: c.buildEndpoint(JaegerThriftHTTPPort), TLSSetting: c.buildTLSSetting()},
		}}
	case ZipkinID:
		return &ServerConfig{
			Endpoint:   c.buildEndpoint(ZipkinPort),
			TLSSetting: c.buildTLSSetting(),
		}
	case StatsdID:
		return map[string]any{
			"endpoint": c.buildEndpoint(StatsdPort),
			"timer_histogram_mapping": []TimerHistogramMapping{
				{
					StatsDType: "histogram", ObserverType: "histogram", Histogram: HistogramConfig{MaxSize: 10},
				},
				{
					StatsDType: "timing", ObserverType: "histogram", Histogram: HistogramConfig{MaxSize: 100},
				},
				{
					StatsDType: "distribution", ObserverType: "histogram", Histogram: HistogramConfig{MaxSize: 100},
				},
			},
		}
	}

	return nil
}

func (c *Config) buildReceivers(protocols []string) (map[component.ID]component.Config, error) {
	if len(protocols) == 0 {
		// means all protocols are enabled
		protocols = []string{string(StatsdProtocol), string(ZipkinProtocol), string(JaegerProtocol), string(OtlpProtocol)}
	}

	receivers := make(map[component.ID]component.Config)

	for _, p := range protocols {
		switch Protocol(p) {
		case StatsdProtocol:
			receivers[StatsdID] = c.buildReceiverComponent(StatsdID)
		case ZipkinProtocol:
			receivers[ZipkinID] = c.buildReceiverComponent(ZipkinID)
		case JaegerProtocol:
			receivers[JaegerID] = c.buildReceiverComponent(JaegerID)
		case OtlpProtocol:
			receivers[OtlpID] = c.buildReceiverComponent(OtlpID)
		default:
			return nil, errors.Errorf("unknown protocol: %s", p)
		}
	}

	return receivers, nil
}
