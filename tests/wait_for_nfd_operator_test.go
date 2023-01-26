package tests

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/client-go/rest"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("wait_for_nfd_operator :", Ordered, func() {
	var (
		config         *rest.Config
		namespace      string
		nfdOperatorCsv *operatorsv1alpha1.ClusterServiceVersion
	)

	BeforeAll(func() {
		config = internal.GetClientConfig()
	})

	It("NFD Operator Should Be Installed successfully", func() {
		succeeded := operatorsv1alpha1.ClusterServiceVersionPhase("Succeeded")
		csvs, err := ocputils.GetCsvsByLabel(config, "", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(csvs.Items).ToNot(BeEmpty())
		for _, csv := range csvs.Items {
			if strings.Contains(csv.Name, "nfd") {
				nfdOperatorCsv = &csv
				namespace = csv.Namespace
			}
		}
		Expect(nfdOperatorCsv).ToNot(BeNil(), "CSV not found")
		if nfdOperatorCsv.Status.Phase != succeeded {
			err = testutils.ExecWithRetryBackoff("Wait for CSV to be Succeeded", func() bool {
				csv, err := ocputils.GetCsvByName(config, nfdOperatorCsv.Namespace, nfdOperatorCsv.Name)
				if err != nil {
					return false
				}
				nfdOperatorCsv = csv
				return nfdOperatorCsv.Status.Phase == succeeded
			}, 15, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(nfdOperatorCsv.Status.Phase).To(Equal(succeeded), "CSV Phase is not Succeeded")
		testutils.Printf("Info", "GPU Operator namespace = %v", namespace)
		csvJson, err := json.MarshalIndent(nfdOperatorCsv, "", " ")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveToArtifactsDir(csvJson, "nfd_operator_csv.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should have a NFD label on a node", func() {
		nfdLabel := "nfd.node.kubernetes.io/feature-labels"
		node, err := ocputils.GetFirstWorkerNode(config)
		Expect(err).ToNot(HaveOccurred())
		_, ok := node.Annotations[nfdLabel]
		Expect(ok).To(BeTrue())
		err = testutils.SaveAsJsonToArtifactsDir(node, "first_worker_node.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("capture namespace", func() {
		ns, err := ocputils.GetNamespace(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(ns, "namespace.json")
		Expect(err).ToNot(HaveOccurred())
	})
})
