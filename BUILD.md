# Build

## Prerequisites

Here is the list of required prerequisites, please make sure you have these tools installed on your system:

- [golang](https://go.dev/dl/)
- [minikube](https://github.com/kubernetes/minikube/releases)
- [docker](https://docs.docker.com/get-docker/)
- [virtualbox](https://www.virtualbox.org/)

You may use latest available versions.

## Instructions

How to build step by step.

Spin up minimal Kubernetes cluster by using Minikube, by default it will use VirtualBox as a virtualization driver

```bash
scripts/minikube.sh
```

Inject environment variables to be able to connect Docker CLI to Kubernetes CRI, which by default is Docker run-time

```bash
eval "$(minikube docker-env)"
```

Now compile the code and bundle as Docker container image, Docker run-time images cache will be populated with the resulting container image

```bash
scripts/build.sh
```

Roll-out deployment

```bash
scripts/apply.sh
```
