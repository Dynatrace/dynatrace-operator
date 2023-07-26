doc/permissions: manifests
	python3 ./hack/doc/role-permissions2md.py ./config/deploy/openshift/openshift-all.yaml > permissions.md
