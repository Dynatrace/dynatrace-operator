
debug/prepare/csi-server:
	git apply hack/make/debug/csi-server.patch

debug/prepare/csi-provisioner:
	git apply hack/make/debug/csi-provisioner.patch

debug/tunnel/csi-driver:
	kubectl port-forward -n dynatrace $(kubectl get pod -n dynatrace -l app.kubernetes.io/component=csi-driver -o jsonpath='{.items[0].metadata.name}') 40000:40000

debug/remove/csi-server:
	git apply -R hack/make/debug/csi-server.patch

debug/remove/csi-provisioner:
	git apply -R hack/make/debug/csi-provisioner.patch
