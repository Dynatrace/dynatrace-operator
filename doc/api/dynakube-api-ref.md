## DynaKube schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiUrl`|Dynatrace `apiUrl`, including the `/api` path at the end. - For SaaS, set `YOUR_ENVIRONMENT_ID` to your environment ID. - For Managed, change the `apiUrl` address. For instructions on how to determine the environment ID and how to configure the apiUrl address, see Environment ID (<https://www.dynatrace.com/support/help/get-started/monitoring-environment/environment-id>).|-|string|
|`customPullSecret`|Defines a custom pull secret in case you use a private registry when pulling images from the Dynatrace environment. To define a custom pull secret and learn about the expected behavior, see Configure customPullSecret (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#custompullsecret>).|-|string|
|`dynatraceApiRequestThreshold`|Minimum minutes between Dynatrace API requests.|15|integer|
|`enableIstio`|When enabled, and if Istio is installed on the Kubernetes environment, Dynatrace Operator will create the corresponding VirtualService and ServiceEntry objects to allow access to the Dynatrace Cluster from the OneAgent or ActiveGate. Disabled by default.|-|boolean|
|`networkZone`|Sets a network zone for the OneAgent and ActiveGate pods.|-|string|
|`proxy`|Set custom proxy settings either directly or from a secret with the field `proxy`. Applies to Dynatrace Operator, ActiveGate, and OneAgents.|-|object|
|`skipCertCheck`|Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster. Set to `true` if you want to skip certification validation checks.|-|boolean|
|`tokens`|Name of the secret holding the tokens used for connecting to Dynatrace.|-|string|
|`trustedCAs`|Adds custom RootCAs from a configmap. The key to the data must be `certs`. This applies to both the Dynatrace Operator and the OneAgent. Doesn't apply to ActiveGate.|-|string|

### .spec.oneAgent

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`hostGroup`|Specify the name of the group to which you want to assign the host. This method is preferred over the now obsolete `--set-host-group` argument. If both settings are used, this field takes precedence over the `--set-host-group` argument.|-|string|

### .spec.activeGate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the ActiveGate pods|-|object|
|`capabilities`|Defines the ActiveGate pod capabilities Possible values: 	- `routing` enables OneAgent routing. 	- `kubernetes-monitoring` enables Kubernetes API monitoring. 	- `metrics-ingest` opens the metrics ingest endpoint on the DynaKube ActiveGate and redirects all pods to it. 	- `dynatrace-api` enables calling the Dynatrace API via ActiveGate.|-|array|
|`customProperties`|Add a custom properties file by providing it as a value or reference it from a secret If referenced from a secret, make sure the key is called `customProperties`|-|object|
|`dnsPolicy`|Sets DNS Policy for the ActiveGate pods|-|string|
|`env`|List of environment variables to set for the ActiveGate|-|array|
|`group`|Set activation group for ActiveGate|-|string|
|`image`|Use a custom ActiveGate image. Defaults to the latest ActiveGate image provided by the registry on the tenant|-|string|
|`labels`|Your defined labels for ActiveGate pods in order to structure workloads as desired.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes ActiveGate will be deployed.|-|object|
|`priorityClassName`|Assign a priority class to the ActiveGate pods. By default, no class is set. For details, see Pod Priority and Preemption. (<https://dt-url.net/n8437bl>)|-|string|
|`replicas`|Amount of replicas for your ActiveGates|1|integer|
|`resources`|Resource settings for ActiveGate container. Consumption of the ActiveGate heavily depends on the workload to monitor. Adjust values accordingly.|-|object|
|`tlsSecretName`|The name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used. `server.p12`: certificate+key pair in pkcs12 format `password`: passphrase to read server.p12|-|string|
|`tolerations`|Set tolerations for the ActiveGate pods|-|array|
|`topologySpreadConstraints`|Adds TopologySpreadConstraints to the ActiveGate pods|-|array|

### .spec.metadataEnrichment

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`enabled`|Enables MetadataEnrichment, `true` by default.|True|boolean|
|`namespaceSelector`|The namespaces where you want Dynatrace Operator to inject enrichment.|-|object|

### .spec.oneAgent.hostMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer. For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>). For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>). Enabled by default.|True|boolean|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR. - `resource.requests` shows the values needed to run - `resource.limits` shows the maximum limits for the pod|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set. For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|`version`|The OneAgent version to be used for host monitoring OneAgents running in the dedicated pod. This setting doesn't affect the OneAgent version used for application monitoring.|-|string|

### .spec.oneAgent.classicFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer. For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>). For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>). Enabled by default.|True|boolean|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR. - `resource.requests` shows the values needed to run - `resource.limits` shows the maximum limits for the pod|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set. For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|`version`|The OneAgent version to be used for host monitoring OneAgents running in the dedicated pod. This setting doesn't affect the OneAgent version used for application monitoring.|-|string|

### .spec.oneAgent.cloudNativeFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer. For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>). For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>). Enabled by default.|True|boolean|
|`codeModulesImage`|The OneAgent image that is used to inject into Pods.|-|string|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|`initResources`|Define resources requests and limits for the initContainer. For details, see Managing resources for containers (<https://kubernetes.io/docs/concepts/configuration/manage-resources-containers>).|-|object|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`namespaceSelector`|Applicable only for applicationMonitoring or cloudNativeFullStack configuration types. The namespaces where you want Dynatrace Operator to inject. For more information, see Configure monitoring for namespaces and pods (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#annotate>).|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR. - `resource.requests` shows the values needed to run - `resource.limits` shows the maximum limits for the pod|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set. For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|`version`|The OneAgent version to be used for host monitoring OneAgents running in the dedicated pod. This setting doesn't affect the OneAgent version used for application monitoring.|-|string|

### .spec.oneAgent.applicationMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`codeModulesImage`|The OneAgent image that is used to inject into Pods.|-|string|
|`initResources`|Define resources requests and limits for the initContainer. For details, see Managing resources for containers (<https://kubernetes.io/docs/concepts/configuration/manage-resources-containers>).|-|object|
|`namespaceSelector`|Applicable only for applicationMonitoring or cloudNativeFullStack configuration types. The namespaces where you want Dynatrace Operator to inject. For more information, see Configure monitoring for namespaces and pods (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#annotate>).|-|object|
|`useCSIDriver`|Set if you want to use the CSIDriver. Don't enable it if you do not have access to Kubernetes nodes or if you lack privileges.|False|boolean|
|`version`|The OneAgent version to be used.|-|string|
