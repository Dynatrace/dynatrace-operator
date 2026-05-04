OAPI_CONFIG_DIR := api/oapi
OAPI_SYNC_CONFIG := $(OAPI_CONFIG_DIR)/sync-config.yaml
OAPI_GENERATOR_CONFIG := $(OAPI_CONFIG_DIR)/generator-config.yaml
OAPI_IGNORE_FILE := $(OAPI_CONFIG_DIR)/.openapi-generator-ignore
OAPI_MODELS_FOR_API := hack/make/bin/oapi-models-for-api.sh

# Defaults from generator-config.yaml (lazily evaluated, only when target runs)
OAPI_GENERATOR_VERSION ?= $(shell yq -r '.generatorVersion' $(OAPI_GENERATOR_CONFIG))
OAPI_OUTPUT_DIR ?= $(shell yq -r '.outputDir' $(OAPI_GENERATOR_CONFIG))
OAPI_ADDITIONAL_PROPS ?= $(shell yq -r '.additionalProperties // ""' $(OAPI_GENERATOR_CONFIG))
OAPI_GLOBAL_PROPS ?= $(shell yq -r '.globalProperties // ""' $(OAPI_GENERATOR_CONFIG))

# Builds the --global-property value from per-schema and global config
jq_global_props = .generate.globalProperties as $$gp \
  | (["$(OAPI_GLOBAL_PROPS)"] | map(select(length > 0))) + [ \
      (if $$gp.apis       and ($$gp.apis       | length) > 0 then "apis="   + ($$gp.apis   | join(":")) else "apis,models" end), \
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
		props=$$(echo "$$row" | jq -r '.generate.additionalProperties // "$(OAPI_ADDITIONAL_PROPS)"'); \
		out=$$(echo "$$row" | jq -r '.generate.outputDir // "$(OAPI_OUTPUT_DIR)/'"$$pkg"'"'); \
		spec_url=$$(echo "$$row" | jq -r '.specUrlEnvVar // ""' | xargs -I{} printenv {} 2>/dev/null || echo ""); \
		auth_var=$$(echo "$$row" | jq -r '.authEnvVar // ""'); \
		auth_val=$$([ -n "$$auth_var" ] && printenv "$$auth_var" || echo ""); \
		auth_header=$$([ -n "$$auth_val" ] && printf 'Authorization:Bearer%%20%s' "$$auth_val" || echo ""); \
		[ -n "$$spec_url" ] || { echo "WARNING: no specUrlEnvVar set for $$name, skipping."; continue; }; \
		tmp_spec=$$(mktemp); \
		if [ -n "$$auth_val" ]; then \
			curl -sSfL --location-trusted -H "Authorization: Bearer $$auth_val" "$$spec_url" -o "$$tmp_spec"; \
		else \
			curl -sSfL "$$spec_url" -o "$$tmp_spec"; \
		fi; \
		apis=$$(echo "$$row" | jq -r '.generate.globalProperties.apis // [] | .[]'); \
		models=""; \
		for api in $$apis; do \
			api_models=$$($(OAPI_MODELS_FOR_API) "$$tmp_spec" "$$api" | tr '\n' ' '); \
			models="$$models $$api_models"; \
		done; \
		models=$$(echo "$$models" | tr ' ' '\n' | sort -u | grep -v '^$$' | tr '\n' ':' | sed 's/:$$//'); \
		gprops=$$(echo "$$row" | jq -r '$(jq_global_props)'); \
		gprops=$$(printf '%s' "$${models:+models=$$models$${gprops:+,}}$$gprops"); \
		echo "Generating $$name (go $$ver, package: $$pkg)..."; \
		rm -rf "$$out" && mkdir -p "$$out"; \
		cp "$(OAPI_IGNORE_FILE)" "$$out/.openapi-generator-ignore"; \
		OPENAPI_GENERATOR_VERSION="$$ver" $(OPENAPI_GENERATOR_CLI) generate \
			-i "$$tmp_spec" -g go -o "$$out" \
			$${pkg:+--package-name "$$pkg"} \
			$${props:+--additional-properties="$$props"} \
			$${gprops:+--global-property="$$gprops"} \
			$${auth_header:+--auth "$$auth_header"} \
			--skip-validate-spec \
			--minimal-update; \
		rm -f "$$tmp_spec" "$$out/.openapi-generator-ignore"; \
		echo "Done: $$out"; \
	done
	@rm -f openapitools.json
