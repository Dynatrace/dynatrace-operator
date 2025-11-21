## Trigger all automatic generated docs creation
.PHONY:
doc: doc/api-ref doc/gen-gomarkdoc

## Generate API docs for custom resources
doc/api-ref: manifests prerequisites/python
	source ./bin/.venv/bin/activate && $(PYTHON) ./hack/doc/custom_resource_params_to_md.py ./config/crd/bases/dynatrace.com_dynakubes.yaml > ./doc/api/dynakube-api-ref.md
	source ./bin/.venv/bin/activate && $(PYTHON) ./hack/doc/custom_resource_params_to_md.py ./config/crd/bases/dynatrace.com_edgeconnects.yaml > ./doc/api/edgeconnect-api-ref.md

## Create a table containing permissions needed by Operator components
doc/permissions: manifests prerequisites/python
	source ./bin/.venv/bin/activate && $(PYTHON) ./hack/doc/role-permissions2md.py ./config/deploy/openshift/openshift-csi.yaml > permissions.md

## Run scripts that generate markdown documentation using gomarkdoc (./hack/doc)
doc/gen-gomarkdoc: prerequisites/gomarkdoc prerequisites/markdownlint
	./hack/doc/gen_e2e_features.sh
	./hack/doc/gen_feature_flags.sh
