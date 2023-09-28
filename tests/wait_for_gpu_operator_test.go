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

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("wait_for_gpu_operator :", Ordered, func() {
	var (
		config         *rest.Config
		namespace      string
		gpuOperatorCsv *operatorsv1alpha1.ClusterServiceVersion
		clusterpolicy  *gpuv1.ClusterPolicy
	)

	BeforeAll(func() {
		config = internal.GetClientConfig()
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
		err := testutils.ExecWithRetryBackoff("Wait for nvidia-operator-validator daemonset", func() bool {
			ds, err := ocputils.GetDaemonset(config, namespace, "nvidia-operator-validator")
			if err != nil || ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
				return false
			}
			dsJson, err := json.MarshalIndent(ds, "", " ")
			Expect(err).ToNot(HaveOccurred())
			err = testutils.SaveToArtifactsDir(dsJson, "nvidia-operator-validator-ds.json")
			Expect(err).ToNot(HaveOccurred())
			Expect(ds.Status.NumberReady).To(Equal(ds.Status.DesiredNumberScheduled), "Validator DS is not ready.")
			return true
		}, 20, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
	})

	It("GPU nodes should be labeled with nvidia.com/gpu.present=true", func() {
		err := testutils.ExecWithRetryBackoff("Wait for nodes with 'nvidia.com/gpu.present=true' label", func() bool {
			resp, err := ocputils.GetNodesByLabel(config, "nvidia.com/gpu.present=true")
			if err != nil {
				return false
			}
			if len(resp.Items) > 0 {
				testutils.Printf("Info", "found #%v nodes with 'nvidia.com/gpu.present=true'", len(resp.Items))
				_ = testutils.SaveAsJsonToArtifactsDir(resp.Items, "gpu_label_found.json")
				return true
			}
			return false
		}, 20, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
	})

	It("GPU nodes should have GPU capacity", func() {
		err := testutils.ExecWithRetryBackoff("Wait for nodes with nvidia.com/gpu.present label to have GPU capacity", func() bool {
			resp, err := ocputils.GetNodesByLabel(config, "nvidia.com/gpu.present=true")
			if err != nil || len(resp.Items) == 0 {
				return false
			}
			val, ok := resp.Items[0].Status.Capacity["nvidia.com/gpu"]
			if !ok {
				return false
			}

			testutils.Printf("Info", "found capacity 'nvidia.com/gpu=%s' on node %s", val.String(), resp.Items[0].Name)
			i, ok := val.AsInt64()
			if !ok {
				testutils.Printf("Error", "unexpected value of nvidia.com/gpu capacity: %s", val.String())
				return false
			}
			if i > 0 {
				_ = testutils.SaveAsJsonToArtifactsDir(resp.Items[0], "gpu_capacity_found.json")
				return true
			}
			return false
		}, 20, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
	})

	It("capture namespace", func() {
		ns, err := ocputils.GetNamespace(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(ns, "namespace.json")
		Expect(err).ToNot(HaveOccurred())
	})
})
