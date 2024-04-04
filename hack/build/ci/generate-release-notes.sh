#!/bin/bash

tag=${GITHUB_REF_NAME:-<tag>}
tag_without_prerelease=${tag%%-*}
tag_without_leading_v=${tag:1}
output_file=${OUTPUT_FILE:-CHANGELOG.md}
pre_release=${PRE_RELEASE:-false}

pre_release_warning="> ⚠️ This is a pre-release, which has no official support by Dynatrace. If you run into issues with this specific release, please open a Github Issue!
>
> Release notes for ${tag_without_prerelease} will be published in our official documentation.
"

pre_release_footer="### Features

<features-go-here>


_Full changelog will be published with the final release, including bugfixes and further smaller improvements!_"

release_footer="### What's Changed
Release Notes can be found in our [official Documentation](https://docs.dynatrace.com/docs/whats-new/release-notes/dynatrace-operator)."

kubernetes_manifests="kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/download/${tag}/kubernetes.yaml"
openshift_manifests="oc apply -f https://github.com/Dynatrace/dynatrace-operator/releases/download/${tag}/openshift.yaml"

kubernetes_manifests="${kubernetes_manifests}
kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/download/${tag}/kubernetes-csi.yaml"

openshift_manifests="${openshift_manifests}
oc apply -f https://github.com/Dynatrace/dynatrace-operator/releases/download/${tag}/openshift-csi.yaml"

if [ "${pre_release}" = false ] ; then
  footer="${release_footer}"
else
  footer="${pre_release_footer}"
fi

footer=""

if [ "${pre_release}" = false ] ; then
  footer="${release_footer}"
else
  footer="${pre_release_footer}"
fi

default_notes="### Installation

For information on how to install the [latest dynatrace-operator](https://github.com/Dynatrace/dynatrace-operator/releases/latest) please visit our [official Documentation](https://docs.dynatrace.com/docs/setup-and-configuration/setup-on-k8s/installation).


#### Helm (recommended)
\`\`\`sh
helm upgrade dynatrace-operator oci://public.ecr.aws/dynatrace/dynatrace-operator \\
  --version ${tag_without_leading_v} \\
  --create-namespace --namespace dynatrace \\
  --install \\
  --atomic
\`\`\`

<details>
  <summary>Other upgrade/install instructions</summary>

#### Kubernetes
\`\`\`sh
${kubernetes_manifests}
\`\`\`

#### Openshift
\`\`\`sh
${openshift_manifests}
\`\`\`

</details>

${footer}"

rm -f "${output_file}"
if [ "$pre_release" = true ] ; then
  echo "$pre_release_warning" >> "${output_file}"
fi

echo "$default_notes" >> "${output_file}"
