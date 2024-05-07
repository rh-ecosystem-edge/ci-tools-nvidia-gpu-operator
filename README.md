# CI Tools for NVIDIA GPU Operator

**WARNING:** This repository is deprecated. The new repository for NVIDIA GPU testing is [nvidia-ci](https://github.com/rh-ecosystem-edge/nvidia-ci).

---

This repository contains a collection of tests for the NVIDIA GPU Operator.
Including setup functions to prepare an OpenShift cluster for testing purposes.

### Basic usage

```shell
#  deploy latest version from certified-operators
$ make deploy_gpu_operator
# deploy a specific channel from certified-operators
$ make deploy_gpu_operator CHANNEL=v1.10
# run E2E test. deploy GPU operator from certified-operators and test operation
$ make e2e_gpu_test
# scale gpu machine set
$ make scale_aws_gpu_nodes [REPLICAS=1 INSTANCE_TYPE=g4dn.xlarge]
# run e2e test on a gpu operator bundle
$ make bundle_e2e_gpu_test BUNDLE=my_bundle.to/test:latest

```
