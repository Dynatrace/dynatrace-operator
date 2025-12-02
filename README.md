# Dynatrace Operator

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/Dynatrace/dynatrace-operator)
[![CI](https://github.com/Dynatrace/dynatrace-operator/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/Dynatrace/dynatrace-operator/actions/workflows/ci.yaml)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/Dynatrace/dynatrace-operator?color=blue&sort=semver)
[![Go Report Card](https://goreportcard.com/badge/github.com/Dynatrace/dynatrace-operator)](https://goreportcard.com/report/github.com/Dynatrace/dynatrace-operator?dummy=unused)
[![Releases](https://img.shields.io/github/downloads/Dynatrace/dynatrace-operator/total.svg)](https://github.com/Dynatrace/dynatrace-operator/releases)

The Dynatrace Operator supports rollout and lifecycle management of various Dynatrace components in Kubernetes and OpenShift. hone test

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
our [official help page](https://www.dynatrace.com/support/help/shortlink/kubernetes).

## Support lifecycle

As the Dynatrace Operator is provided by Dynatrace Incorporated, support is provided by the Dynatrace Support team, as described on the [support page](https://support.dynatrace.com/).
Github issues will also be considered on a case-by-case basis regardless of support contracts and commercial relationships with Dynatrace.

The [Dynatrace support lifecycle for Kubernetes and Openshift](https://www.dynatrace.com/support/help/shortlink/support-model-k8s-ocp) can be found in the official technology support pages.

## Quick Start

The Dynatrace Operator acts on its separate namespace `dynatrace`. It holds the operator deployment and all dependent
objects like permissions, custom resources and corresponding StatefulSets.

### Installation

> For install instructions, head to the
> [official help page](https://www.dynatrace.com/support/help/shortlink/kubernetes)

## Hacking

See [HACKING](HACKING.md) for details on how to get started enhancing Dynatrace Operator.

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting changes.

## License

Dynatrace Operator is under Apache 2.0 license. See [LICENSE](LICENSE) for details.

## Reporting Issues or Ideas

If you find a bug or security issue, please report it to Dynatrace support by [creating a ticket](https://support.dynatrace.com/).
If you have an idea or feature request, please join our [Dynatrace Community](https://community.dynatrace.com) and create a post.
