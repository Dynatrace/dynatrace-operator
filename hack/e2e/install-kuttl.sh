#!/bin/bash
if ! kubectl kuttl; then
  sudo curl -Lo /usr/local/bin/kubectl-kuttl https://github.com/kudobuilder/kuttl/releases/download/v0.11.1/kubectl-kuttl_0.11.1_linux_x86_64
  sudo chmod +x /usr/local/bin/kubectl-kuttl
fi
