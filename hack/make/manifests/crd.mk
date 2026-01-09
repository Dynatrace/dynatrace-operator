## Generates a CRD in config/crd/bases
manifests/crd/generate: prerequisites/controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

## Generates a CRD in config/crd and then applies it to a cluster using kubectl
manifests/crd/install: prerequisites/kustomize manifests/crd/generate
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

## Generates a CRD in config/crd to remove it from a cluster using kubectl
manifests/crd/uninstall: prerequisites/kustomize manifests/crd/generate
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

## Builds a CRD and puts it with the Helm charts
manifests/crd/helm: prerequisites/kustomize helm/version manifests/crd/generate
	./hack/helm/generate-crd.sh $(KUSTOMIZE) $(HELM_CRD_DIR) $(MANIFESTS_DIR)

## Builds a CRD for the release
manifests/crd/release: manifests/crd/helm
	helm template dynatrace-operator config/helm/chart/default \
			--namespace dynatrace \
			--no-hooks \
			--set manifests=true \
			--show-only templates/Common/crd/*.yaml > $(RELEASE_CRD_YAML)
