package tests

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("test_gpu_operator_metrics :", Ordered, func() {
	var (
		config            *rest.Config
		gpuOperatorCsv    *operatorsv1alpha1.ClusterServiceVersion = nil
		dcgmPods          []corev1.Pod
		kubeconfig        string
		namespace         string
		dcgmPodServerPort string
	)

	BeforeAll(func() {
		kubeconfig = internal.Config.KubeconfigPath
		dcgmPodServerPort = "9400"

		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		Expect(err).ToNot(HaveOccurred())
	})

	It("capture GPU Operator namespace", func() {
		csvs, err := ocputils.GetCsvsByLabel(config, "", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(csvs.Items).ToNot(BeEmpty())
		for _, csv := range csvs.Items {
			if strings.Contains(csv.Name, "gpu-operator-certified") {
				gpuOperatorCsv = &csv
				namespace = csv.Namespace
			}
		}
		Expect(gpuOperatorCsv).ToNot(BeNil(), "CSV not found")
		Expect(namespace).ToNot(BeEmpty())
	})

	It("check if the GPU Operator namespace has the openshift.io/cluster-monitoring label", func() {
		ns, err := ocputils.GetNamespace(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		val, ok := ns.Labels["openshift.io/cluster-monitoring"]
		Expect(ok).To(BeTrue(), "Namespace has no label openshift.io/cluster-monitoring")
		Expect(val).To(Equal("true"), "openshift.io/cluster-monitoring label value is not true")
		err = testutils.SaveAsJsonToArtifactsDir(ns, "gpu_operator_namespace.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("validate that the DCGM metrics are correctly exposed", func() {
		pods, err := ocputils.GetPodsByLabel(config, namespace, "app=nvidia-dcgm-exporter")
		Expect(err).ToNot(HaveOccurred())
		dcgmPods = pods.Items
		podsReady := true
		for _, pod := range pods.Items {
			podsReady = pod.Status.Phase == corev1.PodRunning && podsReady
			podFileName := fmt.Sprintf("pod_%v.json", pod.Name)
			err := testutils.SaveAsJsonToArtifactsDir(pod, podFileName)
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(podsReady).To(BeTrue(), "One or more DCGM exporters is not ready")
	})

	It("wait for DCGM exporter logs to show valid state", func() {
		podStates := map[string]bool{}
		err := testutils.ExecWithRetryBackoff("Waiting for valid output", func() bool {
			for _, pod := range dcgmPods {
				if val, ok := podStates[pod.Name]; ok && val {
					continue
				}
				output_resp, err := ocputils.GetPodLogs(config, pod, false)
				if err != nil {
					return false
				}
				output := *output_resp
				filename := fmt.Sprintf("pod_%v_output.log", pod.Name)
				_ = testutils.SaveToArtifactsDir([]byte(output), filename)
				match1 := strings.Contains(output, "DCGM successfully initialized!")
				match2 := strings.Contains(output, "Kubernetes metrics collection enabled!")
				match3 := strings.Contains(output, "Starting webserver")
				podStates[pod.Name] = match1 && match2 && match3
			}
			if len(podStates) != len(dcgmPods) {
				return false
			}
			for _, ok := range podStates {
				if !ok {
					return false
				}
			}
			return true
		}, 15, 30*time.Second)

		Expect(err).ToNot(HaveOccurred(), "Not all DCGM exporters are ready")
	})

	It("check the DCGM is exporting scrape pool", func() {
		pod := dcgmPods[0]
		resp, err := ocputils.PodProxyGet(config, pod, dcgmPodServerPort, "metrics", map[string]string{})
		Expect(err).ToNot(HaveOccurred())
		Expect(string(resp)).ToNot(BeEmpty(), "scrape pool is empty")
		outputFileName := fmt.Sprintf("%v_metrics_respose.txt", pod.Name)
		err = testutils.SaveToArtifactsDir(resp, outputFileName)
		Expect(err).ToNot(HaveOccurred())
	})

	It("check that prometheus is picking up DCGM service monitor", func() {
		prometheusSecret, err := ocputils.GetSecret(config, "openshift-monitoring", "prometheus-k8s")
		Expect(err).ToNot(HaveOccurred())
		secretVal, err := ocputils.GetSecretValue(prometheusSecret, "prometheus.yaml.gz", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.Contains(*secretVal, "(nvidia-dcgm-exporter);true")).To(BeTrue(), "Prometheus is not picking up DCGM metrics")
	})

})
