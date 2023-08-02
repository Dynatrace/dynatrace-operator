#!/bin/bash

tag=${GITHUB_REF_NAME:-<tag>}
tag_without_prerelease=${tag%%-*}
output_file=${OUTPUT_FILE:-CHANGELOG.md}
pre_release=${PRE_RELEASE:-false}

pre_release_warning="> ⚠️ This is a pre-release, which has no official support by Dynatrace. If you run into issues with this specific release, please open a Github Issue!
> 
> Release notes for ${tag_without_prerelease} will be published in our official documentation.
"

default_notes="### Installation

For information on how to install the [latest dynatrace-operator](https://github.com/Dynatrace/dynatrace-operator/releases/latest) please visit our [official Documentation](https://www.dynatrace.com/support/help/shortlink/full-stack-dto-k8).

<details>
  <summary>Upgrade/Install instructions</summary>

#### Kubernetes
\`\`\`sh
kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/download/${tag}/kubernetes.yaml
\`\`\`

#### Openshift
\`\`\`sh
oc apply -f https://github.com/Dynatrace/dynatrace-operator/releases/download/${tag}/openshift.yaml
\`\`\`

</details>

### Features

<features-go-here>


_Full changelog will be published with the final release, including bugfixes and further smaller improvements!_"

rm -f $output_file
if [ "$pre_release" = true ] ; then
  echo "$pre_release_warning" >> $output_file
fi

echo "$default_notes" >> $output_file
