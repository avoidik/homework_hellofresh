#!/bin/bash

#
# set virtualization driver for minikube
#
VM_DRIVER="virtualbox" # docker

#
# create minikube dev server
#
minikube start \
    --container-runtime docker \
    --driver "$VM_DRIVER" \
    --cpus 2 \
    --memory 2048 \
    --disk-size 8GB \
    --kubernetes-version v1.21.11

#
# dns workaround for virtualbox
#
if [[ "$VM_DRIVER" == "virtualbox" ]]; then
  minikube stop
  VBoxManage modifyvm "minikube" --natdnshostresolver1 off
  VBoxManage modifyvm "minikube" --natdnsproxy1 on
  minikube start
fi
