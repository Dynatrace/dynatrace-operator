**Note: Requires Dynatrace Cluster version 1.209**

# Dynatrace Operator

The Dynatrace Operator supports rollout and lifecycle management of various Dynatrace components in Kubernetes and OpenShift.

* OneAgent
  * `classicFullStack` rolls out a OneAgent pod per node to monitor pods on it and the node itself
  * `applicationMonitoring` is a webhook based injection mechanism for automatic app-only injection
    * (PREVIEW) CSI Driver can be enabled to cache OneAgent downloads per node
  * `hostMonitoring` is only monitoring the hosts (i.e. nodes) in the cluster without app-only injection
  * (PREVIEW) `cloudNativeFullStack` is a combination of `applicationMonitoring` with CSI driver and `hostMonitoring`
* ActiveGate
  * `routing` routes OneAgent traffic through the ActiveGate
  * `kubernetes-monitoring` allows monitoring of the Kubernetes API
  * (PREVIEW) `metrics-ingest` routes enriched metrics through ActiveGate

For more information please have a look at [our DynaKube Custom Resource examples](config/samples) and
our [official help page](https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/).

## Supported platforms

Depending on the version of the Dynatrace Operator, it supports the following platforms:

| Dynatrace Operator version | Kubernetes | OpenShift Container Platform |
|----------------------------|------------|------------------------------|
| master                     | 1.21+      | 4.7+                         |
| v0.4.0                     | 1.21+      | 4.7+                         |
| v0.3.0                     | 1.20+      | 4.7+                         |
| v0.2.2                     | 1.18+      | 3.11.188+, 4.5+              |
| v0.1.0                     | 1.18+      | 3.11.188+, 4.4+              |

## Quick Start

The Dynatrace Operator acts on its separate namespace `dynatrace`. It holds the operator deployment and all dependent
objects like permissions, custom resources and corresponding StatefulSets.

### Installation

> For install instructions on Openshift, head to the
> [official help page](https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/openshift/set-up-ocp-monitoring#set-up-openshift-monitoring)

To create the namespace and apply the operator run the following commands

```sh
$ kubectl create namespace dynatrace
$ kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
```

If using `cloudNativeFullStack` or `applicationMonitoring` with CSI driver, the following command is required as well:
```sh
$ kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes-csi.yaml
```

A secret holding tokens for authenticating to the Dynatrace cluster needs to be created upfront. Create access tokens of
type *Dynatrace API* and use its values in the following commands respectively. For
assistance please refer
to [Create user-generated access tokens](https://www.dynatrace.com/support/help/get-started/access-tokens#create-api-token).

Make sure the tokens have the following permissions:
* API Token
  * Read Configuration
  * Write Configuration
  * Read Entities (if using automatic kubernetes api monitoring)
  * Installer Download
  * Access problem and event feed, metrics and topology
* Data Ingest Token
  * Ingest Metrics

```sh
$ kubectl -n dynatrace create secret generic dynakube --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="dataIngestToken=DATA_INGEST_TOKEN"
```

#### Create `DynaKube` custom resource for ActiveGate and OneAgent rollout

The rollout of the Dynatrace components is governed by a custom resource of type `DynaKube`. This custom resource will
contain parameters for various Dynatrace capabilities (OneAgent deployment mode, ActiveGate capabilities, etc.)

> Note: `.spec.tokens` denotes the name of the secret holding access tokens.
>
> If not specified Dynatrace Operator searches for a secret called like the DynaKube custom resource `.metadata.name`.

The recommended approach is using classic Fullstack injection to roll out Dynatrace to your cluster, available as [classicFullStack sample](config/samples/classicFullStack.yaml).
In case you want to have adjustments please have a look at [our DynaKube Custom Resource examples](config/samples).

Save one of the sample configurations, change the API url to your environment and apply it to your cluster.
```sh
$ kubectl apply -f cr.yaml
```

For detailed instructions see
our [official help page](https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/).


## Uninstall dynatrace-operator

> For instructions on how to uninstall the dynatrace-operator on Openshift, head to the [official help page](https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/openshift/set-up-ocp-monitoring#uninstall-dynatrace-operator)

Clean-up all Dynatrace Operator specific objects:
```sh
$ kubectl delete -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
```

If the CSI driver was installed, the following command is required as well:
```sh
$ kubectl delete -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes-csi.yaml
```

## Hacking

See [HACKING](HACKING.md) for details on how to get started enhancing Dynatrace Operator.

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting changes.

## License

Dynatrace Operator is under Apache 2.0 license. See [LICENSE](LICENSE) for details.
