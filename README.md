# Dynatrace Operator

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/Dynatrace/dynatrace-operator)
[![CI](https://github.com/Dynatrace/dynatrace-operator/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/Dynatrace/dynatrace-operator/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/Dynatrace/dynatrace-operator/parse/branch/main/graph/badge.svg)](https://codecov.io/gh/Dynatrace/dynatrace-operator)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/Dynatrace/dynatrace-operator?color=blue&sort=semver)
[![Go Report Card](https://goreportcard.com/badge/github.com/Dynatrace/dynatrace-operator)](https://goreportcard.com/report/github.com/Dynatrace/dynatrace-operator?dummy=unused)
[![Releases](https://img.shields.io/github/downloads/Dynatrace/dynatrace-operator/total.svg)](https://github.com/Dynatrace/dynatrace-operator/releases)

The Dynatrace Operator supports rollout and lifecycle management of various Dynatrace components in Kubernetes and OpenShift.

* OneAgent
  * `classicFullStack` rolls out a OneAgent pod per node to monitor pods on it and the node itself
  * `applicationMonitoring` is a webhook based injection mechanism for automatic app-only injection
    * CSI Driver can be enabled to cache OneAgent downloads per node
  * `hostMonitoring` is only monitoring the hosts (i.e. nodes) in the cluster without app-only injection
    * CSI Driver is used to provide a writeable volume for the Oneagent as it's running in read-only mode
  * `cloudNativeFullStack` is a combination of `applicationMonitoring` and `hostMonitoring`
    * CSI Driver is used for both features
* ActiveGate
  * `routing` routes OneAgent traffic through the ActiveGate
  * `kubernetes-monitoring` allows monitoring of the Kubernetes API
  * `metrics-ingest` routes enriched metrics through ActiveGate

For more information please have a look at [our DynaKube Custom Resource examples](assets/samples/dynakube) and
our [official help page](https://www.dynatrace.com/support/help/shortlink/kubernetes-hub).

## Support lifecycle

As the Dynatrace Operator is provided by Dynatrace Incorporated, support is provided by the Dynatrace Support team, as described on the [support page](https://support.dynatrace.com/).
Github issues will also be considered on a case-by-case basis regardless of support contracts and commercial relationships with Dynatrace.

The [Dynatrace support lifecycle for Kubernetes and Openshift](https://www.dynatrace.com/support/help/shortlink/support-model-k8s-ocp) can be found in the official technology support pages.

## Quick Start

The Dynatrace Operator acts on its separate namespace `dynatrace`. It holds the operator deployment and all dependent
objects like permissions, custom resources and corresponding StatefulSets.

### Installation

> For install instructions on Openshift, head to the
> [official help page](https://www.dynatrace.com/support/help/shortlink/full-stack-dto-k8)

First, create the namespace

```sh
kubectl create namespace dynatrace
```

Then install the operator with **one** of the following commands.
If you are using `cloudNativeFullStack` or `applicationMonitoring` the CSI Driver is required.

Use this commmand to install the operator **without** CSI driver
```sh
kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
```

Use this command to install the operator **with CSI driver**.
```sh
kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes-csi.yaml
```
**Do not use both commands together**. If you need to add CSI driver later, it is required to first uninstall the operator and then
reinstall it with the CSI driver.

A secret holding tokens for authenticating to the Dynatrace cluster needs to be created upfront. Create access tokens of
type *Dynatrace API* and use its values in the following commands respectively. For
assistance please refer
to [Create user-generated access tokens](https://www.dynatrace.com/support/help/shortlink/token#create-api-token).

The token scopes required by the Dynatrace Operator are documented on our [official help page](https://www.dynatrace.com/support/help/shortlink/full-stack-dto-k8#tokens)

```sh
kubectl -n dynatrace create secret generic dynakube --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="dataIngestToken=DATA_INGEST_TOKEN"
```

#### Create `DynaKube` custom resource for ActiveGate and OneAgent rollout

The rollout of the Dynatrace components is governed by a custom resource of type `DynaKube`. This custom resource will
contain parameters for various Dynatrace capabilities (OneAgent deployment mode, ActiveGate capabilities, etc.)

> Note: `.spec.tokens` denotes the name of the secret holding access tokens.
>
> If not specified Dynatrace Operator searches for a secret called like the DynaKube custom resource `.metadata.name`.

The recommended approach is using classic Fullstack injection to roll out Dynatrace to your cluster, available as [classicFullStack sample](assets/samples/dynakube/v1beta2/classicFullStack.yaml).
In case you want to have adjustments please have a look at [our DynaKube Custom Resource examples](assets/samples/dynakube).

Save one of the sample configurations, change the API url to your environment and apply it to your cluster.

```sh
kubectl apply -f cr.yaml
```

For detailed instructions see
our [official help page](https://www.dynatrace.com/support/help/shortlink/full-stack-dto-k8).

## Uninstall dynatrace-operator

> For instructions on how to uninstall the dynatrace-operator on Openshift,
> head to the [official help page](https://docs.dynatrace.com/docs/setup-and-configuration/setup-on-k8s/guides/operation/update-uninstall-operator#uninstall-dynatrace-operator)

Clean-up all Dynatrace Operator specific objects:

```sh
kubectl delete -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
```

If the CSI driver was installed, the following command is required as well:

```sh
kubectl delete -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes-csi.yaml
```

## Hacking

See [HACKING](HACKING.md) for details on how to get started enhancing Dynatrace Operator.

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting changes.

## License

Dynatrace Operator is under Apache 2.0 license. See [LICENSE](LICENSE) for details.
