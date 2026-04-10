OAPI_CONFIG_DIR := api/oapi
OAPI_SYNC_CONFIG := $(OAPI_CONFIG_DIR)/sync-config.yaml
OAPI_GENERATOR_CONFIG := $(OAPI_CONFIG_DIR)/generator-config.yaml
OAPI_IGNORE_FILE := $(OAPI_CONFIG_DIR)/.openapi-generator-ignore

# Defaults from generator-config.yaml (lazily evaluated, only when target runs)
OAPI_GENERATOR ?= $(shell yq -r '.generator' $(OAPI_GENERATOR_CONFIG))
OAPI_GENERATOR_VERSION ?= $(shell yq -r '.generatorVersion' $(OAPI_GENERATOR_CONFIG))
OAPI_OUTPUT_DIR ?= $(shell yq -r '.outputDir' $(OAPI_GENERATOR_CONFIG))
OAPI_ADDITIONAL_PROPS ?= $(shell yq -r '.additionalProperties' $(OAPI_GENERATOR_CONFIG))

# Derive git owner/repo from the remote for generated import paths
OAPI_GIT_REMOTE ?= $(shell git remote get-url origin 2>/dev/null \
	| sed -E 's%https://github.com/%%' \
	| sed -E 's%git@github.com:%%' \
	| sed -E 's%\.git$$%%')
OAPI_GIT_USER_ID ?= $(firstword $(subst /, ,$(OAPI_GIT_REMOTE)))
OAPI_GIT_REPO_ID ?= $(lastword $(subst /, ,$(OAPI_GIT_REMOTE)))

## Generate Go SDKs from OpenAPI specs
oapi/generate: prerequisites/openapi-generator-cli
	@yq -o=json '.schemas | map(select(.generate))' "$(OAPI_SYNC_CONFIG)" \
	| jq -c '.[]' \
	| while read -r row; do \
		name=$$(echo "$$row" | jq -r '.name'); \
		pkg=$$(echo "$$row" | jq -r '.generate.packageName // .name'); \
		ver=$$(echo "$$row" | jq -r '.generate.generatorVersion // "$(OAPI_GENERATOR_VERSION)"'); \
		gen=$$(echo "$$row" | jq -r '.generate.generator // "$(OAPI_GENERATOR)"'); \
		props=$$(echo "$$row" | jq -r '.generate.additionalProperties // "$(OAPI_ADDITIONAL_PROPS)"'); \
		out=$$(echo "$$row" | jq -r '.generate.outputDir // "$(OAPI_OUTPUT_DIR)/'"$$name"'"'); \
		spec="$(OAPI_CONFIG_DIR)/$$name/spec3.json"; \
		[ -f "$$spec" ] || { echo "WARNING: $$spec not found, skipping $$name."; continue; }; \
		echo "Generating $$name ($$gen $$ver, package: $$pkg)..."; \
		rm -rf "$$out" && mkdir -p "$$out"; \
		cp "$(OAPI_IGNORE_FILE)" "$$out/.openapi-generator-ignore"; \
		OPENAPI_GENERATOR_VERSION="$$ver" $(OPENAPI_GENERATOR_CLI) generate \
			-i "$$spec" -g "$$gen" -o "$$out" \
			--package-name "$$pkg" \
			--additional-properties="$$props" \
			--git-user-id "$(OAPI_GIT_USER_ID)" \
			--git-repo-id "$(OAPI_GIT_REPO_ID)/$$out" \
			--skip-validate-spec; \
		rm -rf "$$out/test" "$$out/api" "$$out/docs"; \
		find "$$out" -name '*_test.go' -delete; \
		echo "Done: $$out"; \
	done
