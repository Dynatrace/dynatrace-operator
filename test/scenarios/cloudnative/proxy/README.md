# Prerequisites

k8s cluster

cilium service mesh

# Setup
CloudNative deployment with CSI driver

# Goals
Verification that the operator and all deployed oneagents are able to communicate over a http proxy.

- DTO installed
- proxy installed
- dynakube with proxy installed
- check if oneagent started
- check if Dynakube phase changed to Running
- check if oneagents are able to communicate with the cluster
- connectivity in the dynatrace namespace is restricted to the local cluster
- connectivity in the sample application namespace is restricted to the local cluster
- check if DT_PROXY environment variable is defined in the dynakube-oneagent container
- sample application deployment is installed
- check if DT_PROXY environment variable is defined in the application container
