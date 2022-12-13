# Prerequisites
k8s cluster

# Setup
CloudNative deployment with CSI driver

# Goals
Verification if ActiveGate is rolled out successfully. All AG capabilities are enabled in Dynakube.

Tests:
  - AG pod has required init containers and containers
  - appropriate Gateway modules are active
  - appropriate mount-points are created in all containers
  - Gateway is reachable via AG service
  - Gateway is in RUNNING state (queried via REST endpoint)

