## ActiveGate schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|annotations|Adds additional annotations to the ActiveGate pods|-|object|
|apiUrl|Dynatrace apiUrl, including the /api path at the end. For SaaS, set YOUR_ENVIRONMENT_ID to your environment ID. For Managed, change the apiUrl address. For instructions on how to determine the environment ID and how to configure the apiUrl address, see Environment ID (<https://www.dynatrace.com/support/help/get-started/monitoring-environment/environment-id>).|-|string|
|capabilities|Activegate capabilities enabled (routing, kubernetes-monitoring, metrics-ingest, dynatrace-api)|-|array|
|customProperties|Add a custom properties file by providing it as a value or reference it from a secret If referenced from a secret, make sure the key is called 'customProperties'|-|object|
|dnsPolicy|Sets DNS Policy for the ActiveGate pods|-|string|
|env|List of environment variables to set for the ActiveGate|-|array|
|group|Set activation group for ActiveGate|-|string|
|image|The ActiveGate container image. Defaults to the latest ActiveGate image provided by the registry on the tenant|-|string|
|labels|Adds additional labels for the ActiveGate pods|-|object|
|networkZone|Sets a network zone for the OneAgent and ActiveGate pods.|-|string|
|nodeSelector|Node selector to control the selection of nodes|-|object|
|priorityClassName|If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that name. If not specified the setting will be removed from the StatefulSet.|-|string|
|proxy|Set custom proxy settings either directly or from a secret with the field proxy. Note: Applies to Dynatrace Operator, ActiveGate, and OneAgents.|-|object|
|replicas|Amount of replicas for your ActiveGates|-|integer|
|resources|Define resources requests and limits for single ActiveGate pods|-|object|
|skipCertCheck|Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster. Set to true if you want to skip certification validation checks.|-|boolean|
|tlsSecretName|The name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used. server.p12: certificate+key pair in pkcs12 format password: passphrase to read server.p12|-|string|
|tokens|Name of the secret holding the tokens used for connecting to Dynatrace.|-|string|
|tolerations|Set tolerations for the ActiveGate pods|-|array|
|topologySpreadConstraints|Adds TopologySpreadConstraints for the ActiveGate pods|-|array|
