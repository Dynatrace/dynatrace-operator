
# How to setup OpenTelementry

Currently only the webhook is instrumented using OpenTelementry. To connect the webhook to a tenant do the following steps:

## Create an access token with the following scopes

- openTelemetryTrace.ingest
- metrics.ingest
- logs.ingest

## Create OpenTelementry configuration secret

```yaml
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: dynatrace-operator-otel-config
  namespace: dynatrace
data:
  endpoint: base64(<uuid>.dev.dyntracelabs.com)
  apiToken: base64(<apiToken>)
```

*Note:*

- as indicated the values have to be base64 encoded (as usually with K8S)
- obey to the name
- make sure it is created in the same namespace as the webhook
