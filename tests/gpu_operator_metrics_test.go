package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
		gpuOpVersion      *version.OperatorVersion
		monitoringLabel   string
	)

	BeforeAll(func() {
		kubeconfig = internal.Config.KubeconfigPath
		dcgmPodServerPort = "9400"
		monitoringLabel = "openshift.io/cluster-monitoring"

		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		Expect(err).ToNot(HaveOccurred())
	})

	It("capture GPU Operator namespace and version", func() {
		csvs, err := ocputils.GetCsvsByLabel(config, "", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(csvs.Items).ToNot(BeEmpty())
		for _, csv := range csvs.Items {
			if strings.Contains(csv.Name, "gpu-operator-certified") {
				gpuOperatorCsv = &csv
				namespace = csv.Namespace
				gpuOpVersion = &csv.Spec.Version
				break
			}
		}
		Expect(gpuOperatorCsv).ToNot(BeNil(), "CSV not found")
		Expect(namespace).ToNot(BeEmpty())
		Expect(gpuOpVersion).ToNot(BeNil())
		err = testutils.SaveAsJsonToArtifactsDir(gpuOperatorCsv, "gpu-operator-csv.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("ensure namespace has the openshift.io/cluster-monitoring label (for versions <= 1.9)", func() {
		versionWithoutLabel := semver.Version{Major: 1, Minor: 9, Patch: 1}
		testutils.Printf("Info", "GPU operator version: %v", gpuOpVersion)
		if gpuOpVersion.Version.GT(versionWithoutLabel) {
			msg := fmt.Sprintf("Installed version is %v, namespace should have the lable already - skipping test", gpuOpVersion.String())
			Skip(msg)
			return
		}
		ns, err := ocputils.GetNamespace(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		filenameBase := "gpu-operator-namespace"
		fileNameBefore := fmt.Sprintf("%s-before-label-patch.json", filenameBase)
		fileNameAfter := fmt.Sprintf("%s-after-label-patch.json", filenameBase)
		err = testutils.SaveAsJsonToArtifactsDir(ns, fileNameBefore)
		Expect(err).ToNot(HaveOccurred())
		ns.ObjectMeta.Labels[monitoringLabel] = "true"
		jsonLabels, err := json.Marshal(ns.ObjectMeta.Labels)
		Expect(err).ToNot(HaveOccurred())
		patch := fmt.Sprintf("{\"metadata\": {\"labels\": %v}}", string(jsonLabels))
		ns, err = ocputils.PatchNamespace(config, ns.Name, []byte(patch), types.MergePatchType)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(ns, fileNameAfter)
		Expect(err).ToNot(HaveOccurred())
	})

	It("check if the GPU Operator namespace has the openshift.io/cluster-monitoring label", func() {
		ns, err := ocputils.GetNamespace(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		val, ok := ns.Labels[monitoringLabel]
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
		serviceMonitor := fmt.Sprintf("job_name: serviceMonitor/%v/nvidia-dcgm-exporter", namespace)
		err := testutils.ExecWithRetryBackoff("DCGM prometheus pickup", func() bool {
			prometheusSecret, err := ocputils.GetSecret(config, "openshift-monitoring", "prometheus-k8s")
			if err != nil {
				testutils.Printf("Error", "%v", err)
				return false
			}
			promyml, err := ocputils.GetSecretValue(prometheusSecret, "prometheus.yaml.gz", true)
			if err != nil {
				testutils.Printf("Error", "%v", err)
				return false
			}
			_ = testutils.SaveToArtifactsDir([]byte(*promyml), "prometheus.yaml.gz.txt")
			return strings.Contains(*promyml, serviceMonitor)
		}, 30, 30*time.Second)
		Expect(err).ToNot(HaveOccurred(), "Prometheus is not picking up DCGM metrics")
	})

})
