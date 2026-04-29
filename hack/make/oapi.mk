OAPI_CONFIG_DIR := api/oapi
OAPI_SYNC_CONFIG := $(OAPI_CONFIG_DIR)/sync-config.yaml
OAPI_GENERATOR_CONFIG := $(OAPI_CONFIG_DIR)/generator-config.yaml
OAPI_IGNORE_FILE := $(OAPI_CONFIG_DIR)/.openapi-generator-ignore

# Defaults from generator-config.yaml (lazily evaluated, only when target runs)
OAPI_GENERATOR ?= $(shell yq -r '.generator' $(OAPI_GENERATOR_CONFIG))
OAPI_GENERATOR_VERSION ?= $(shell yq -r '.generatorVersion' $(OAPI_GENERATOR_CONFIG))
OAPI_OUTPUT_DIR ?= $(shell yq -r '.outputDir' $(OAPI_GENERATOR_CONFIG))
OAPI_ADDITIONAL_PROPS ?= $(shell yq -r '.additionalProperties // ""' $(OAPI_GENERATOR_CONFIG))
OAPI_GLOBAL_PROPS ?= $(shell yq -r '.globalProperties // ""' $(OAPI_GENERATOR_CONFIG))

# Builds the --global-property value by merging OAPI_GLOBAL_PROPS (global default)
# with per-schema globalProperties from sync-config.yaml.
# Global is prepended; per-schema is appended — duplicate keys are resolved by last-value-wins.
# Per-schema accepts a YAML object with 'models', 'apis', and 'additional' fields, or a plain string.
jq_global_props = .generate.globalProperties as $$gp \
  | (["$(OAPI_GLOBAL_PROPS)"] | map(select(length > 0))) + [ \
      (if $$gp.models     and ($$gp.models     | length) > 0 then "models=" + ($$gp.models | join(":")) else empty end), \
      (if $$gp.apis       and ($$gp.apis       | length) > 0 then "apis="   + ($$gp.apis   | join(":")) else empty end), \
      (if $$gp.additional and ($$gp.additional | length) > 0 then $$gp.additional                       else empty end) \
    ] | join(",")


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
		gprops=$$(echo "$$row" | jq -r '$(jq_global_props)'); \
		out=$$(echo "$$row" | jq -r '.generate.outputDir // "$(OAPI_OUTPUT_DIR)/'"$$pkg"'"'); \
		repo_var=$$(echo "$$row" | jq -r '.repoEnvVar // ""'); \
		repo_url=$$([ -n "$$repo_var" ] && printenv "$$repo_var" || echo ""); \
		spec_path=$$(echo "$$row" | jq -r '.specPath // "spec3.json"'); \
		spec=$$([ -n "$$repo_url" ] && echo "$$repo_url/$$spec_path" || echo "$(OAPI_CONFIG_DIR)/$$name/$$spec_path"); \
		auth_var=$$(echo "$$row" | jq -r '.authEnvVar // ""'); \
		auth_val=$$([ -n "$$auth_var" ] && printenv "$$auth_var" || echo ""); \
		[ -n "$$repo_url" ] || [ -f "$$spec" ] || { echo "WARNING: $$spec not found, skipping $$name."; continue; }; \
		echo "Generating $$name ($$gen $$ver, package: $$pkg)..."; \
		rm -rf "$$out" && mkdir -p "$$out"; \
		cp "$(OAPI_IGNORE_FILE)" "$$out/.openapi-generator-ignore"; \
		OPENAPI_GENERATOR_VERSION="$$ver" $(OPENAPI_GENERATOR_CLI) generate \
			-i "$$spec" -g "$$gen" -o "$$out" \
			$${pkg:+--package-name "$$pkg"} \
			$${props:+--additional-properties="$$props"} \
			$${gprops:+--global-property="$$gprops"} \
			$${auth_val:+--auth "Authorization:Bearer $$auth_val"} \
			--skip-validate-spec \
			--minimal-update; \
		rm -rf "$$out/.openapi-generator-ignore"; \
		echo "Done: $$out"; \
	done
