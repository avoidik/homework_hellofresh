#!/bin/bash

#
# deploy application to Kubernetes cluster, requires valid Kubernetes config context
#

kubectl apply -f k8s/manifest.yaml
