package setup

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("test_ocp_connection :", Ordered, func() {
	kubeconfigPath := internal.Config.KubeconfigPath

	It("should have a valid kubeconfig path", func() {
		testutils.Printf("Info", "Using KUBECONFIG=%v", kubeconfigPath)
		Expect(kubeconfigPath).ToNot(BeEmpty())
		_, err := os.Stat(kubeconfigPath)
		Expect(err).ToNot(HaveOccurred(), "Failed to fetch file info of kubeconfig")
	})

	It("Should successfuly get server versions", func() {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		Expect(err).ToNot(HaveOccurred())
		serverVersion, err := ocputils.GetServerVersion(config)
		Expect(err).ToNot(HaveOccurred())
		ocpVersionMsg := fmt.Sprintf("K8s Version: %v\nOCP Version: %v\n", serverVersion.Kubernetes, serverVersion.Openshift)
		err = testutils.SaveToArtifactsDir([]byte(ocpVersionMsg), "OCP_Version.txt")
		Expect(err).ToNot(HaveOccurred())
		testutils.Printf("Info", ocpVersionMsg)
	})
})
