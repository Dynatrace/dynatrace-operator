# Prerequisites
k8s cluster

# Setup
ApplicationMonitoring deployment without CSI driver

# Goals
### Verification of the data ingest part of the operator
Scenarios:
- sample application Deployment installed
  - check if enrichment variables added to the initContainer
  - check if dt_metadata.json file contains required fields
- sample application Pod installed
  - enrichment variables added to the initContainer
  - dt_metadata.json file contains required fields

### Verification of build label propagation
Scenarios:
- default behavior
  - feature flag exists, but no additional configuration so the default variables are added
- custom mapping
  - feature flag exists, with additional configuration so all 4 build variables are added
- preserved values of existing variables
  - build variables exist, feature flag exists, with additional configuration, values of build variables not get overwritten
- incorrect custom mapping
  - invalid name of BUILD VERSION label, reference exists but actual label doesn't exist
