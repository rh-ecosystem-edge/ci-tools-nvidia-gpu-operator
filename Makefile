# Usage: make deploy_gpu_operator [CHANNEL=v1.10]
.DEFAULT_GOAL := default

.PHONY: test_ocp_connection
test_ocp_connection:
	@./hack/run_test.sh test_ocp_connection

.PHONY: deploy_nfd_operator
deploy_nfd_operator:
	@./hack/run_test.sh deploy_nfd_operator

.PHONY: deploy_gpu_operator
deploy_gpu_operator:
	@./hack/run_test.sh deploy_gpu_operator $(CHANNEL)

.PHONY: clean_artifact_dir
clean_artifact_dir:
	@./hack/run_test.sh clean_artifact_dir

.PHONY: wait_for_gpu_operator
wait_for_gpu_operator:
	@./hack/run_test.sh wait_for_gpu_operator

.PHONY: run_gpu_workload
run_gpu_workload:
	@./hack/run_test.sh run_gpu_workload

.PHONY: check_exported_metrics
check_exported_metrics:
	@./hack/run_test.sh check_exported_metrics

.PHONY: wait_for_nfd_operator
wait_for_nfd_operator:
	@./hack/run_test.sh wait_for_nfd_operator

.PHONY: test_gpu_operator_metrics
test_gpu_operator_metrics:
	@./hack/run_test.sh test_gpu_operator_metrics

.PHONY: e2e_gpu_test
e2e_gpu_test: deploy_gpu_operator wait_for_gpu_operator run_gpu_workload test_gpu_operator_metrics

.PHONY: scale_aws_gpu_nodes
scale_aws_gpu_nodes:
	@INSTANCE_TYPE=$(INSTANCE_TYPE) REPLICAS=$(REPLICAS) ./hack/run_test.sh scale_aws_gpu_nodes


.PHONY: lint
lint:
	golangci-lint run -v


.PHONY: default
default:
	@echo "No Target Selected"; exit 1
