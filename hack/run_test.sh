#!/bin/bash

THIS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

export ARTIFACT_DIR=${ARTIFACT_DIR:="/tmp/gpu-test"}
export KUBECONFIG=${KUBECONFIG:="$HOME/.kube/config"}
export GPU_OPERATOR_MASTER_BUNDLE="registry.gitlab.com/nvidia/kubernetes/gpu-operator/staging/gpu-operator-bundle:master-latest"

#####################
## Setup functions ##
#####################
function test_ocp_connection() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    (ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./setup/ && create_dashboard_ocp_version "${ART_DIR}") || error_and_exit "${FUNCNAME[0]} Test Failed." 1
}

function deploy_nfd_operator() {
    print_test_title "${FUNCNAME[0]}"
    test_ocp_connection
    echo
    echo "=> Deploying NFD Operator"
    echo
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
	ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./setup/ || error_and_exit "${FUNCNAME[0]} Test Failed." 2
}

function deploy_gpu_operator() {
    print_test_title "${FUNCNAME[0]}"
    test_ocp_connection
    export GPU_CHANNEL="$1"
    echo
    echo "=> Deploying GPU Op. Channel ${GPU_CHANNEL}"
    echo
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
	ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./setup/ || error_and_exit "${FUNCNAME[0]} Test Failed." 3
}

function scale_aws_gpu_nodes() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    export GPU_INSTANCE_TYPE="${INSTANCE_TYPE:-}"
    export GPU_REPLICAS="${REPLICAS:-}"
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./setup/ || error_and_exit "${FUNCNAME[0]} Test Failed." 4 "$1"
}

function deploy_gpu_operator_master() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    export DEPLOYED_FROM_MASTER=true
    operator-sdk run bundle --timeout=10m -n "gpu-operator-test" \
        --install-mode OwnNamespace \
        ${GPU_OPERATOR_MASTER_BUNDLE} || error_and_exit "${FUNCNAME[0]} Test Failed" 5
    deploy_gpu_operator "" || error_and_exit "${FUNCNAME[0]} Test Failed" 6
}

function ocm_addons_setup() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./setup/ || error_and_exit "${FUNCNAME[0]} Test Failed." 6
}

#####################
## Test  functions ##
#####################

function wait_for_gpu_operator() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    (ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./tests/ && create_dashboard_operator_version "${ART_DIR}") || error_and_exit "${FUNCNAME[0]} Test Failed." 11
}

function run_gpu_workload() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./tests/ || error_and_exit "${FUNCNAME[0]} Test Failed." 12
}

function check_exported_metrics() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./tests/ || error_and_exit "${FUNCNAME[0]} Test Failed." 13
}


function wait_for_nfd_operator() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./tests/ || error_and_exit "${FUNCNAME[0]} Test Failed." 14 "$1"
}

function test_gpu_operator_metrics() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./tests/ || error_and_exit "${FUNCNAME[0]} Test Failed." 15
}

function gpu_addon_must_gather() {
    print_test_title "${FUNCNAME[0]}"
    ART_DIR=$(dirgen "${FUNCNAME[0]}")
    GINKGO_ARGS=$(ginko_args "${ART_DIR}" "${FUNCNAME[0]}")
    ARTIFACT_DIR=$ART_DIR ginkgo ${GINKGO_ARGS} ./tests/ || error_and_exit "${FUNCNAME[0]} Test Failed." 16
}

########################
## General  functions ##
########################
function clean_artifact_dir() {
    print_test_title "${FUNCNAME[0]}"
    rm -rf "${ARTIFACT_DIR}"/FAIL
    rm -rf "${ARTIFACT_DIR}"/SUCCESS
    rm -rf "${ARTIFACT_DIR}"/RETURN_CODE
    rm -rf "${ARTIFACT_DIR}"/*.log
    ls -d "${ARTIFACT_DIR}"/* | grep -P "[0-9]{10}_" | xargs  rm -rf 
}


function print_test_title(){
    echo
    echo "==> Running Test: ${1}"
    echo
}

function error_and_exit() {
    local can_fail="$3";

    if [ "$can_fail" != "true" ];
    then
        rm -rf "${ARTIFACT_DIR}/SUCCESS"
        echo "${1}" | tee "${ARTIFACT_DIR}/FAIL"
        echo "${2}" > "${ARTIFACT_DIR}/RETURN_CODE"
    fi
    exit "$2"
}

function dirgen() {
    timestamp=$(date +%s)
    dir="${ARTIFACT_DIR}/${timestamp}_${1}"
    mkdir -p "$dir"
    echo "$dir"
}

function ginko_args() {
    ART_DIR="$1"
    shift
    NAME="$1"
    echo "--output-dir=$ART_DIR --junit-report=junit_report_${NAME}.xml --fail-fast --succinct --no-color --focus $NAME"
}

### Init output folder
OUTPUT_FILE="${ARTIFACT_DIR}/output-${1}.log"
mkdir -p "${ARTIFACT_DIR}"
bash "$THIS_DIR/print_title.sh" | tee "${OUTPUT_FILE}"

## CI Dashboad files
git rev-parse --short HEAD > "${ARTIFACT_DIR}/ci_artifact.git_version"
function create_dashboard_ocp_version() {
    connection_dir=$1
    tail -1 "${connection_dir}/OCP_Version.txt" | tr ' ' '\n' | tail -1 > "${ARTIFACT_DIR}/ocp.version"
}
function create_dashboard_operator_version() {
    wait_for_gpu_operator_dir=$1
    cp "${wait_for_gpu_operator_dir}/gpu_operator_version.txt" "${ARTIFACT_DIR}/operator.version"
}

case "$1" in
#####################
## Setup functions ##
#####################
    test_ocp_connection) "$@" | tee -a "${OUTPUT_FILE}";;
    deploy_nfd_operator) "$@" | tee -a "${OUTPUT_FILE}";;
    deploy_gpu_operator) "$@" | tee -a "${OUTPUT_FILE}";;
    scale_aws_gpu_nodes) "$@" | tee -a "${OUTPUT_FILE}";;
    deploy_gpu_operator_master) "$@" | tee -a "${OUTPUT_FILE}";;
    ocm_addons_setup) "$@" | tee -a "${OUTPUT_FILE}";;

#####################
## Test  functions ##
#####################
    wait_for_gpu_operator) "$@" | tee -a "${OUTPUT_FILE}";;
    wait_for_nfd_operator) "$@" | tee -a "${OUTPUT_FILE}";;
    test_gpu_operator_metrics) "$@" | tee -a "${OUTPUT_FILE}";;
    run_gpu_workload) "$@" | tee -a "${OUTPUT_FILE}";;
    check_exported_metrics) "$@" | tee -a "${OUTPUT_FILE}";;
    gpu_addon_must_gather) "$@" | tee -a "${OUTPUT_FILE}";;

    clean_artifact_dir) "$@";exit;;
    *) error_and_exit "Invalid operation $1." 44 | tee -a "${OUTPUT_FILE}";;
esac

if [[ ! -f "${ARTIFACT_DIR}/FAIL" ]]; then
    echo "SUCCESS" > "${ARTIFACT_DIR}/SUCCESS"
    echo 0 > "${ARTIFACT_DIR}/RETURN_CODE"
else
    exit $(cat "${ARTIFACT_DIR}/RETURN_CODE")
fi



