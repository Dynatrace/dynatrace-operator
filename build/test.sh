#!/bin/bash

########## Prepare directories for Kubebuilder ##########
sudo mkdir -p /usr/local/kubebuilder
sudo chmod 777 /usr/local/kubebuilder

########## Fetch Kubebuilder ##########
if [ ! -d "/usr/local/kubebuilder/bin" ]; then
    curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.3.1/kubebuilder_2.3.1_linux_amd64.tar.gz -o kubebuilder.tar.gz
    tar -zxvf kubebuilder.tar.gz --strip-components=1 -C /usr/local/kubebuilder
fi

########## Run tests ##########
go test -cover -v ./...

########## Run integration tests ##########
go test -cover -tags integration,containers_image_storage_stub -v ./...
