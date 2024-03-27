# OpenTelemetry in the operator

*Note:* This guide is still work-in-progress and will evolve as we learn more about OpenTelemetry instrumentation best practices.

## How to setup OpenTelementry

Dynatrace operator, CSI driver and webhook are instrumented using OpenTelemetry. To enable this instrumentation and ingest collected
metrics and traces into your tenant follow this guide.

### Create an access token with the following scopes

- openTelemetryTrace.ingest
- metrics.ingest
- logs.ingest

### Create OpenTelementry configuration secret

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: dynatrace-operator-otel-config
  namespace: dynatrace
data:
  endpoint: base64(<uuid>.dev.dyntracelabs.com)
  apiToken: base64(<apiToken>)
EOF
```

*Note:*

- as indicated the values have to be base64 encoded (as usually with K8S)
- obey to the name
- make sure it is created in the same namespace as the webhook

### How to use it in E2E tests

Create a file at `test/testdata/secrets/otel-tenant.yaml`, according to `test/testdata/secrets-samples/otel-tenant.yaml`.

- It will ask for the same info that you would use in the secret.

## OpenTelemetry instrumentation guidelines

### Spans and Traces

#### Record errors

- Start a span, in fucntions where errors can happen and use that span to record the error(s).
- Errors shall be recorded where they originate
-Do not record errors multiple times, when they bubble up the call chain

#### Start span in expensive functions

- Spans shall be started in functions that are heavy in
  - IO
  - runtime
  - memory
  - complexity
- We don't necessarily need spans in all functions, but we need to identify the important parts of the code and instrument them.
