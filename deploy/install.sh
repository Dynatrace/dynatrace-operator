#!/bin/sh

set -e

bold=$(tput bold)
normal=$(tput sgr0)

echo ${bold}This script is deprecated - installation aborted.${normal}
echo
echo If your tenant references this script, we recommend that you update your tenant to the latest version.
echo This will provide you with updated deployment instructions on the \"Monitor Kubernetes/OpenShift\" page.
echo
echo An alternative approach is to manually set up Kubernetes monitoring.
echo For more information, please refer to the official documentation:
echo https://dt-url.net/deprecated-installation
