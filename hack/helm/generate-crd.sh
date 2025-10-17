#!/usr/bin/env bash

KUSTOMIZE="$1"
HELM_CRD_DIR="$2"
MAINFEST_DIR="$3"

# Create the crd with the conversion webhook patch
DYNATRACE_OPERATOR_CRD_YAML="dynatrace-operator-crd.yaml"
SOURCE_CRD_DIR="${MAINFEST_DIR}/kubernetes"
SOURCE_CRD_FILE="${SOURCE_CRD_DIR}/${DYNATRACE_OPERATOR_CRD_YAML}"

mkdir -p "${HELM_CRD_DIR}"
"${KUSTOMIZE}" build config/crd >"${SOURCE_CRD_FILE}"

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

# Define the header for the helm yaml file
HELM_HEADER="{{ if .Values.installCRD }}"

# Get the previously patched crd content
CRD_CONTENT="$(cat "${SOURCE_CRD_FILE}")"

# Define the helm yaml footer to match the header
HELM_FOOTER="{{- end -}}"

# Overwrite the previously generated CRD
{
	echo "$HELM_HEADER"
	echo "$CRD_CONTENT"
	echo "$HELM_FOOTER"
} >"${HELM_CRD_DIR}/${DYNATRACE_OPERATOR_CRD_YAML}"

rm "${SOURCE_CRD_FILE}"
