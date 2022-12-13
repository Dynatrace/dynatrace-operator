# Prerequisites
k8s cluster

# Setup
DTO with CSI driver

# Goals
Verification if support-archive package created by the support-archive command and printed to standard output is a valid tar(.gz) package.

Scanario:
- DTO installed
- support-archive command executed on the operator pod
- check if the archive package contains "operator-version.txt" file
- check if the whole tar package has correct size

