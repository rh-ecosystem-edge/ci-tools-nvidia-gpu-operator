package tests

import (
	"encoding/json"
	"strings"
	"time"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("wait_for_gpu_operator :", Ordered, func() {
	kubeconfig := internal.Config.KubeconfigPath
	var (
		config         *rest.Config
		namespace      string
		gpuOperatorCsv *operatorsv1alpha1.ClusterServiceVersion
		clusterpolicy  *gpuv1.ClusterPolicy
	)

	BeforeAll(func() {
		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		Expect(err).ToNot(HaveOccurred())
	})

	It("GPU operator should be installed successfully", func() {
		succeeded := operatorsv1alpha1.ClusterServiceVersionPhase("Succeeded")
		csvs, err := ocputils.GetCsvsByLabel(config, "", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(csvs.Items).ToNot(BeEmpty())
		for _, csv := range csvs.Items {
			if strings.Contains(csv.Name, "gpu-operator-certified") {
				gpuOperatorCsv = &csv
				namespace = csv.Namespace
				break
			}
		}
		Expect(gpuOperatorCsv).ToNot(BeNil(), "CSV not found")
		Expect(namespace).ToNot(BeEmpty())
		testutils.Printf("Info", "GPU Operator name=%v namespace=%v version=%v", gpuOperatorCsv.Name, gpuOperatorCsv.Namespace, gpuOperatorCsv.Spec.Version.String())
		if gpuOperatorCsv.Status.Phase != succeeded {
			err = testutils.ExecWithRetryBackoff("Wait for CSV to be Succeeded", func() bool {
				csv, err := ocputils.GetCsvByName(config, gpuOperatorCsv.Namespace, gpuOperatorCsv.Name)
				if err != nil {
					return false
				}
				gpuOperatorCsv = csv
				return gpuOperatorCsv.Status.Phase == succeeded
			}, 15, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(gpuOperatorCsv.Status.Phase).To(Equal(succeeded), "CSV Phase is not Succeeded")
		err = testutils.SaveAsJsonToArtifactsDir(gpuOperatorCsv, "gpu_operator_csv.json")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveToArtifactsDir([]byte(gpuOperatorCsv.Spec.Version.String()), "gpu_operator_version.txt")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should have GPU Nodes", func() {
		labelSelectors := []string{
			"feature.node.kubernetes.io/pci-10de.present",
			"feature.node.kubernetes.io/pci-0302_10de.present",
			"feature.node.kubernetes.io/pci-0300_10de.present",
		}
		err := testutils.ExecWithRetryBackoff("Wait For GPU Nodes", func() bool {
			for _, label := range labelSelectors {
				resp, err := ocputils.GetNodesByLabel(config, label)
				if err != nil {
					return false
				}
				if len(resp.Items) > 0 {
					testutils.Printf("Info", "found #%v GPU nodes", len(resp.Items))
					_ = testutils.SaveAsJsonToArtifactsDir(resp.Items, "gpu_nodes_found.json")
					return true
				}
			}
			return false
		}, 15, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should have a ClusterPolicy", func() {
		resp, err := ocputils.ListDynamicResource(config, gpuv1.GroupVersion.WithResource("clusterpolicies"))
		Expect(err).ToNot(HaveOccurred())
		Expect(len(resp.Items)).To(Equal(1), "ClusterPolicies in this cluster does not equal 1")
		clusterpolicy = &gpuv1.ClusterPolicy{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resp.Items[0].UnstructuredContent(), clusterpolicy)
		Expect(err).ToNot(HaveOccurred())
		testutils.Printf("Info", "ClusterPolicy Found. Name=%v", clusterpolicy.Name)
		cpJson, err := json.MarshalIndent(clusterpolicy, "", " ")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveToArtifactsDir(cpJson, "clusterpolicy.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("nvidia-operator-validator Daemonset should be ready", func() {
		validatorDs, err := ocputils.GetDaemonset(config, namespace, "nvidia-operator-validator")
		Expect(err).ToNot(HaveOccurred())

		if validatorDs.Status.NumberReady != validatorDs.Status.DesiredNumberScheduled {
			err = testutils.ExecWithRetryBackoff("Wait for nvidia-operator-validator daemonset", func() bool {
				ds, err := ocputils.GetDaemonset(config, namespace, "nvidia-operator-validator")
				if err != nil {
					return false
				}
				validatorDs = ds
				return ds.Status.NumberReady == ds.Status.DesiredNumberScheduled
			}, 20, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		}

		dsJson, err := json.MarshalIndent(validatorDs, "", " ")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveToArtifactsDir(dsJson, "nvidia-operator-validator-ds.json")
		Expect(err).ToNot(HaveOccurred())

		Expect(validatorDs.Status.NumberReady).To(Equal(validatorDs.Status.DesiredNumberScheduled), "Validator DS is not ready.")
	})
})
