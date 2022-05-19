#!/bin/bash

installCGODependencies() {
  sudo apt-get update
  sudo apt-get install -y libdevmapper-dev libbtrfs-dev libgpgme-dev
}

installKubebuilderIfNotExists() {
  ########## Prepare directories for Kubebuilder ##########
  sudo mkdir -p /usr/local/kubebuilder
  sudo chmod 777 /usr/local/kubebuilder

  ########## Fetch Kubebuilder ##########
  if [ ! -d "/usr/local/kubebuilder/bin" ]; then
      curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.3.1/kubebuilder_2.3.1_linux_amd64.tar.gz -o kubebuilder.tar.gz
      tar -zxvf kubebuilder.tar.gz --strip-components=1 -C /usr/local/kubebuilder
  fi
}

# call function provided in arguments
"$@"
