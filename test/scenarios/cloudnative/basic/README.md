# Prerequisites
k8s cluster

# Setup
CloudNative deployment with CSI driver

# Goals
Verify that the operator is working as expected.

Scenarios:
### Install
- DTO installed
- sample application Deployment installed
- check if oneagent started
- check if Dynakube phase changed to Running
- sample application restarted
- check if oneagent init container is injected
- check if oneagents are able to communicate with the cluster

### Upgrade
Verify that 0.9.1 released version can be updated to the current version

- DTO "v0.9.1" installed (from GitHub)
- sample application Deployment installed
- check if oneagent started
- check if Dynakube phase changed to Running
- sample application restarted
- check if oneagent init container is injected
- check if oneagents are able to communicate with the cluster
- DTO updated to the current version
- half of sample application pods restarted
- check if oneagent init container is injected
- check if oneagents are able to communicate with the cluster

### CodeModules
Verify that the storage in the CSI driver directory does not increase when there are multiple tenants and pods which are monitored.

- DTO installed
- sample application Deployment installed
- check if oneagent started
- check if Dynakube phase changed to Running
- sample application restarted
- check if oneagent init container is injected
- check if oneagents are able to communicate with the cluster
- check if csi driver is working
- check if codemodules image has been downloaded
- check if storage size has not increased (2 tenants)
- check if volumes are mounted correctly

### Specific Agent Version
Verify that the operator is able to set agent version which is given via the dynakube. Upgrading to a newer version of agent is also tested.

- DTO installed
- sample application deployment installed
- dynakube with agent of specified version installed
- sample application restarted
- check if oneagent init container is injected
- check if VERSION environment variable is correct
- dynakube updated to newer agent version
- sample application restarted
- check if oneagent init container is injected
- check if VERSION environment variable is correct
