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
manifests/crd/helm: helm/version prerequisites/kustomize manifests/crd/generate
	# Build crd
	mkdir -p "$(HELM_CRD_DIR)"
	$(KUSTOMIZE) build config/crd > "$(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)"

	sed "s/namespace: dynatrace/namespace: {{.Release.Namespace}}/" "$(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)" > "$(MANIFESTS_DIR)/kubernetes/tmp_crd"
	mv "$(MANIFESTS_DIR)/kubernetes/tmp_crd" "$(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)"

	echo "{{- include \"dynatrace-operator.platformRequired\" . }}" > "$(HELM_CRD_FILE)"
	echo "{{ if and .Values.installCRD (eq (include \"dynatrace-operator.partial\" .) \"false\") }}" >> "$(HELM_CRD_FILE)"
	cat "$(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)" >> "$(HELM_CRD_FILE)"
	echo "{{- end -}}" >> "$(HELM_CRD_FILE)"

