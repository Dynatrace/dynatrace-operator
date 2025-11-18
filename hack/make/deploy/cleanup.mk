
## Remove all Dynatrace Operator resources from the cluster and node filesystem
cleanup: cleanup/cluster cleanup/node-fs

## Remove all Dynatrace Operator resources from the cluster
cleanup/cluster:
	@./hack/cluster/cleanup-dynatrace-objects.sh

## Remove node filesystem leftovers
cleanup/node-fs:
	@SKIP_RUNNING_PODS_WARNING=true ./hack/cluster/cleanup-node-fs.sh dynatrace
