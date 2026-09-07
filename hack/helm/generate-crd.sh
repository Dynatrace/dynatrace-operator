#!/usr/bin/env bash

KUSTOMIZE="$1"
HELM_CRD_DIR="$2"
MAINFEST_DIR="$3"
KUSTOMIZE_DIR="$4"
OUTPUT_FILENAME="$5"
EXPERIMENTAL_FLAG="$6"

HELM_HEADER='{{ if .Values.installCRD }}'
if [ -n "${EXPERIMENTAL_FLAG}" ]; then
  HELM_HEADER="{{ if and .Values.installCRD (.Values.experimental).${EXPERIMENTAL_FLAG} }}"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOILERPLATE="${SCRIPT_DIR}/../boilerplate.go.txt"

# Convert boilerplate.go.txt (Go // comment) to YAML # comment style
yaml_license_header() {
    awk '{ sub(/^\/\/ /, "# "); print }' "${BOILERPLATE}"
    echo ""
}

# Create the crd with the conversion webhook patch
SOURCE_CRD_DIR="${MAINFEST_DIR}/kubernetes"
SOURCE_CRD_FILE="${SOURCE_CRD_DIR}/${OUTPUT_FILENAME}"

mkdir -p "${HELM_CRD_DIR}"
"${KUSTOMIZE}" build "${KUSTOMIZE_DIR}" >"${SOURCE_CRD_FILE}"

# Replace the the namespace specified in the webhook service to the helm-chart template string
# does not use sed -i, because it's not supported by default in MacOS
sed "s/namespace: dynatrace/namespace: {{.Release.Namespace}}/" "${SOURCE_CRD_FILE}" >"${SOURCE_CRD_DIR}/tmp_crd"
mv "${SOURCE_CRD_DIR}/tmp_crd" "${SOURCE_CRD_FILE}"

# Add the common labels by finding the line 'name: dynakubes.dynatrace.com' and inserting labels before it
awk 'BEGIN{inserted=0} /name: dynakubes.dynatrace.com/ && !inserted {print "  labels:"; print "    {{- include \"dynatrace-operator.commonLabels\" . | nindent 4 }}"; inserted=1} {print}' "${SOURCE_CRD_FILE}" > "${SOURCE_CRD_DIR}/tmp_crd"
mv "${SOURCE_CRD_DIR}/tmp_crd" "${SOURCE_CRD_FILE}"

# Add the common labels by finding the line 'name: edgeconnects.dynatrace.com' and inserting labels before it
awk 'BEGIN{inserted=0} /name: edgeconnects.dynatrace.com/ && !inserted {print "  labels:"; print "    {{- include \"dynatrace-operator.commonLabels\" . | nindent 4 }}"; inserted=1} {print}' "${SOURCE_CRD_FILE}" > "${SOURCE_CRD_DIR}/tmp_crd"
mv "${SOURCE_CRD_DIR}/tmp_crd" "${SOURCE_CRD_FILE}"

# Add the common labels by finding the line 'name: dtprometheuses.dynatrace.com' and inserting labels before it
awk 'BEGIN{inserted=0} /name: dtprometheuses.dynatrace.com/ && !inserted {print "  labels:"; print "    {{- include \"dynatrace-operator.commonLabels\" . | nindent 4 }}"; inserted=1} {print}' "${SOURCE_CRD_FILE}" > "${SOURCE_CRD_DIR}/tmp_crd"
mv "${SOURCE_CRD_DIR}/tmp_crd" "${SOURCE_CRD_FILE}"


# Get the previously patched crd content
CRD_CONTENT="$(cat "${SOURCE_CRD_FILE}")"

# Define the helm yaml footer to match the header
HELM_FOOTER="{{- end -}}"

# Overwrite the previously generated CRD
{
	echo "$HELM_HEADER"
	yaml_license_header
	echo "$CRD_CONTENT"
	echo "$HELM_FOOTER"
} >"${HELM_CRD_DIR}/${OUTPUT_FILENAME}"

rm "${SOURCE_CRD_FILE}"
