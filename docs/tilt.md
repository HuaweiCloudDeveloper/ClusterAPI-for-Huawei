# Developing Cluster API Provider Huawei for with Tilt

## Overview

This document describes how to use [kind](https://kind.sigs.k8s.io) and [Tilt](https://tilt.dev) for a simplified
workflow that offers easy deployments and rapid iterative builds.

## Prerequisites

1. [Docker](https://docs.docker.com/install/): v19.03 or newer (on MacOS e.g. via [ObrStack](https://orbstack.dev/))
2. [kind](https://kind.sigs.k8s.io): v0.25.0 or newer
3. [Tilt](https://docs.tilt.dev/install.html): v0.30.8 or newer
4. [kustomize](https://github.com/kubernetes-sigs/kustomize): provided via `make kustomize`
5. [envsubst](https://github.com/drone/envsubst): provided via `make envsubst`
6. [helm](https://github.com/helm/helm): v3.7.1 or newer
7. Clone the [Cluster API](https://github.com/kubernetes-sigs/cluster-api) repository
   locally
8. Clone the provider(s) you want to deploy locally as well

## Getting started

### Create a tilt-settings file

create a `tilt-settings.yaml` file and place it in your local copy of `cluster-api`. Here is an example that uses the components from the CAPI repo:

```yaml
default_registry: localhost:5000
enable_providers:
- huawei
- kubeadm-bootstrap
- kubeadm-control-plane
provider_repos:
- ../cluster-api-provider-huawei
template_dirs:
  huawei:
  - ../cluster-api-provider-huawei/templates
kustomize_substitutions:
  CLOUD_SDK_AK: "xxxxxxxx"
  CLOUD_SDK_SK: "xxxxxxxx"

  HC_REGION: "cn-north-4"
  HC_SSH_KEY_NAME: "default"
  KUBERNETES_VERSION: "v1.26.15"
  CLUSTER_NAME: "hello"
  CONTROL_PLANE_MACHINE_COUNT: "1"
  WORKER_MACHINE_COUNT: "1"
  HC_CONTROL_PLANE_MACHINE_TYPE: "x1e.2u.4g"
  HC_NODE_MACHINE_TYPE: "x1e.2u.4g"

  ECS_IMAGE_ID: "a9f5cc27-0d50-4864-a45e-d8eead734a3f"
```

**Note:** Please ensure that the values for `CLOUD_SDK_AK` and `CLOUD_SDK_SK` are base64 encoded.

### Create a kind cluster and run Tilt!

To create a pre-configured kind cluster (if you have not already done so) and launch your development environment, run

```bash
make tilt-up
```

---

For details, please refer to: https://cluster-api.sigs.k8s.io/developer/core/tilt
