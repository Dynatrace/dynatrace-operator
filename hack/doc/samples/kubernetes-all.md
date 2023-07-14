
## Dynatrace Kubernetes Monitoring (ActiveGate) (ClusterRole)

|Resources accessed |API group |APIs used |Resource names |
|------------------ |--------- |--------- |-------------- |
|Nodes |`core` |List/Watch/Get | |
|Pods |`core` |List/Watch/Get | |
|Namespaces |`core` |List/Watch/Get | |
|ReplicationControllers |`core` |List/Watch/Get | |
|Events |`core` |List/Watch/Get | |
|ResourceQuotas |`core` |List/Watch/Get | |
|Pods/Proxy |`core` |List/Watch/Get | |
|Nodes/Proxy |`core` |List/Watch/Get | |
|Nodes/Metrics |`core` |List/Watch/Get | |
|Services |`core` |List/Watch/Get | |
|Jobs |`batch` |List/Watch/Get | |
|CronJobs |`batch` |List/Watch/Get | |
|Deployments |`apps` |List/Watch/Get | |
|ReplicaSets |`apps` |List/Watch/Get | |
|StatefulSets |`apps` |List/Watch/Get | |
|DaemonSets |`apps` |List/Watch/Get | |
|DeploymentConfigs |`apps.openshift.io` |List/Watch/Get | |
|ClusterVersions |`config.openshift.io` |List/Watch/Get | |

## Dynatrace Operator (ClusterRole)

|Resources accessed |API group |APIs used |Resource names |
|------------------ |--------- |--------- |-------------- |
|Nodes |`core` |Get/List/Watch | |
|Namespaces |`core` |Get/List/Watch/Update | |
|Secrets |`core` |Create | |
|Secrets |`core` |Get/Update/Delete/List |`dynatrace-dynakube-config`<br />`dynatrace-data-ingest-endpoint`<br />`dynatrace-activegate-internal-proxy` |
|Events |`core` |Create/Patch | |
|MutatingWebhookConfigurations |`admissionregistration.k8s.io` |Get/Update |`dynatrace-webhook` |
|ValidatingWebhookConfigurations |`admissionregistration.k8s.io` |Get/Update |`dynatrace-webhook` |
|CustomResourceDefinitions |`apiextensions.k8s.io` |Get/Update |`dynakubes.dynatrace.com` |

## Dynatrace webhook server (ClusterRole)

|Resources accessed |API group |APIs used |Resource names |
|------------------ |--------- |--------- |-------------- |
|Namespaces |`core` |Get/List/Watch/Update | |
|Events |`core` |Create/Patch | |
|Secrets |`core` |Create | |
|Secrets |`core` |Get/List/Watch/Update |`dynatrace-dynakube-config`<br />`dynatrace-data-ingest-endpoint` |
|ReplicationControllers |`core` |Get | |
|ReplicaSets |`apps` |Get | |
|StatefulSets |`apps` |Get | |
|DaemonSets |`apps` |Get | |
|Deployments |`apps` |Get | |
|Jobs |`batch` |Get | |
|CronJobs |`batch` |Get | |
|DeploymentConfigs |`apps.openshift.io` |Get | |

## Dynatrace Operator (Role)

|Resources accessed |API group |APIs used |Resource names |
|------------------ |--------- |--------- |-------------- |
|Dynakubes |`dynatrace.com` |Get/List/Watch/Update/Create | |
|Dynakubes/Finalizers |`dynatrace.com` |Update | |
|Dynakubes/Status |`dynatrace.com` |Update | |
|StatefulSets |`apps` |Get/List/Watch/Create/Update/Delete | |
|DaemonSets |`apps` |Get/List/Watch/Create/Update/Delete | |
|ReplicaSets |`apps` |Get/List/Watch | |
|Deployments |`apps` |Get/List/Watch | |
|Deployments/Finalizers |`apps` |Update | |
|ConfigMaps |`core` |Get/List/Watch/Create/Update/Delete | |
|Pods |`core` |Get/List/Watch/Delete/Create | |
|Secrets |`core` |Get/List/Watch/Create/Update/Delete | |
|Events |`core` |List/Create | |
|Services |`core` |Create/Update/Delete/Get/List/Watch | |
|Pods/Log |`core` |Get | |
|ServiceMonitors |`monitoring.coreos.com` |Get/Create | |
|ServiceEntries |`networking.istio.io` |Get/List/Create/Update/Delete | |
|VirtualServices |`networking.istio.io` |Get/List/Create/Update/Delete | |
|Leases |`coordination.k8s.io` |Get/Update/Create | |

## Dynatrace webhook server (Role)

|Resources accessed |API group |APIs used |Resource names |
|------------------ |--------- |--------- |-------------- |
|Services |`core` |Get/List/Watch/Create/Update | |
|ConfigMaps |`core` |Get/List/Watch/Create/Update | |
|Secrets |`core` |Get/List/Watch/Create/Update | |
|Pods |`core` |Get/List/Watch | |
|Dynakubes |`dynatrace.com` |Get/List/Watch | |
|Events |`core` |List/Create | |
|Leases |`coordination.k8s.io` |Get/Update/Create | |
|DaemonSets |`apps` |List/Watch | |

## Dynatrace CSI driver (ClusterRole)

|Resources accessed |API group |APIs used |Resource names |
|------------------ |--------- |--------- |-------------- |
|Namespaces |`core` |Get/List/Watch | |
|Events |`core` |List/Watch/Create/Update/Patch | |
|CsiNodes |`storage.k8s.io` |Get/List/Watch | |
|Nodes |`core` |Get/List/Watch | |
|Pods |`core` |Get/List/Watch | |

## Dynatrace CSI driver (Role)

|Resources accessed |API group |APIs used |Resource names |
|------------------ |--------- |--------- |-------------- |
|EndPoints |`core` |Get/Watch/List/Delete/Update/Create | |
|Leases |`coordination.k8s.io` |Get/Watch/List/Delete/Update/Create | |
|Dynakubes |`dynatrace.com` |Get/List/Watch | |
|Secrets |`core` |Get/List/Watch | |
|ConfigMaps |`core` |Get/List/Watch | |
