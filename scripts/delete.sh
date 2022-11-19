#!/bin/bash

#
# deploy application to Kubernetes cluster, requires valid Kubernetes config context
#

kubectl delete -f k8s/manifest.yaml --wait=false
