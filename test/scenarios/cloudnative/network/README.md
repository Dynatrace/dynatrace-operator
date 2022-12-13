# Prerequisites
k8s cluster

cilium service mesh

# Setup
CloudNative deployment with CSI driver

# Goals
Check that the CSI driver is able to recover from network issues, when using cloudNative and code modules image

Scenario:
- connectivity for csi driver pods is restricted to the local k8s cluster (no outside connections allowed)
- sample application is installed
- check if init container was attached and run successful
- check that the sample pods are up and running
