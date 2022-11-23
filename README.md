# CI Tools for NVIDIA GPU Operator

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
$ make scale_aws_gpu_nodes [REPLICAS=1 INSTANCE_TYPE=g4dn.xlarge
```
