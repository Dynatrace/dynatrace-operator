
debug/prepare/csi-server:
	git apply hack/make/debug/csi-server.patch

debug/tunnel/csi-server:
	kubectl port-forward -n dynatrace $(kubectl get pod -n dynatrace -l app.kubernetes.io/component=csi-driver -o jsonpath='{.items[0].metadata.name}') 40000:40000

debug/remove/csi-server:
	git apply -R hack/make/debug/csi-server.patch
