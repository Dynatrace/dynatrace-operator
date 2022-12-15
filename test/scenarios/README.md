### Table of contents
- [Active Gate](#activegate)
- [Application Monitoring](#applicationmonitoring)
  - [data ingest](#data-ingest)
  - [build label propagation](#build-label-propagation)
- [Classic](#classic)
- Cloud Native
  - [basic](#cloudnative---basic)
  - [istio](#cloudnative---istio)
  - [network](#cloudnative---network)
  - [proxy](#cloudnative---proxy)
- [Support Archive](#supportarchive)

# ActiveGate
## Prerequisites

## Setup
CloudNative deployment with CSI driver

## Goals
Verification if ActiveGate is rolled out successfully. All ActiveGate
capabilities are enabled in Dynakube. The test checks if related *Gateway*
modules are active and that the *Gateway* process is reachable via *Gateway service*.

# ApplicationMonitoring
## Prerequisites

## Setup
ApplicationMonitoring deployment without CSI driver

## Goals
### Data Ingest
Verification of the data ingest part of the operator. The test checks that
enrichment variables are added to the initContainer and dt_metadata.json
file contains required fields.

### Build Label Propagation
Verification that build labels are created and set accordingly. The test checks:
- default behavior - feature flag exists, but no additional configuration so the default variables are added
- custom mapping - feature flag exists, with additional configuration so all 4 build variables are added
- preserved values of existing variables - build variables exist, feature flag exists, with additional configuration, values of build variables not get overwritten
- incorrect custom mapping - invalid name of BUILD VERSION label, reference exists but actual label doesn't exist

# Classic
## Prerequisites

## Setup
ClassicFullStack deployment

## Goals
Verification of classic-fullstack deployment. Sample application Deployment is
installed and restarted to check if OneAgent is injected and can communicate
with the *Dynatrace Cluster*.

# CloudNative - basic
## Prerequisites

## Setup
CloudNative deployment with CSI driver

## Goals

### Install
Verification that OneAgent is injected to sample application pods and can
communicate with the *Dynatrace Cluster*.

### Upgrade
Verification that a *released version* can be updated to the *current version*.
The exact number of *released version* is updated manually. The *released
version* is installed using configuration files from GitHub.

Sample application Deployment is installed and restarted to check if OneAgent is
injected and can communicate with the *Dynatrace Cluster*.

### CodeModules
Verification that the storage in the CSI driver directory does not increase when
there are multiple tenants and pods which are monitored.

### Specific Agent Version
Verification that the operator is able to set agent version which is given via
the dynakube. Upgrading to a newer version of agent is also tested.

Sample application Deployment is installed and restarted to check if OneAgent is
injected and VERSION environment variable is correct.

# CloudNative - istio
## Prerequisites
istio service mesh

## Setup
CloudNative deployment with CSI driver

## Goals
Verify that the operator is working as expected when istio service mesh is
installed and pre-configured on the cluster.

1) [Install](#install)
2) [Upgrade](#upgrade)
3) [CodeModules](#codemodules)
4) [Specific Agent Version](#specific-agent-version)

# CloudNative - network
## Prerequisites
cilium service mesh

## Setup
CloudNative deployment with CSI driver

## Goals
Verification that the CSI driver is able to recover from network issues, when
using cloudNative and code modules image.

Connectivity for csi driver pods is restricted to the local k8s cluster (no
outside connections allowed) and sample application is installed. The test
checks if init container was attached, run successfully and that the sample
pods are up and running.

# CloudNative - proxy
## Prerequisites
cilium service mesh

## Setup
CloudNative deployment with CSI driver

## Goals
Verification that the operator and all deployed OneAgents are able to communicate
over a http proxy.

Connectivity in the dynatrace namespace and sample application namespace is restricted to
the local cluster. Sample application is installed. The test checks if DT_PROXY environment
variable is defined in the *dynakube-oneagent* container and the *application container*.

# SupportArchive
## Prerequisites

## Setup
DTO with CSI driver

## Goals
Verification if support-archive package created by the support-archive command and printed
to the standard output is a valid tar.gz package and contains required *operator-version.txt*
file.
