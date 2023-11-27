doc/api-ref: manifests
	python3 ./hack/doc/custom_resource_params_to_md.py ./config/crd/bases/dynatrace.com_dynakubes.yaml > ./doc/dynakube-api-ref.md
	python3 ./hack/doc/custom_resource_params_to_md.py ./config/crd/bases/dynatrace.com_edgeconnects.yaml > ./doc/edgeconnect-api-ref.md
