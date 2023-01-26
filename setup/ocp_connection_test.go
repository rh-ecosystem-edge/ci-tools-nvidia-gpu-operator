package setup

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("test_ocp_connection :", Ordered, func() {
	It("should successfuly get server versions", func() {
		config := internal.GetClientConfig()

		serverVersion, err := ocputils.GetServerVersion(config)
		Expect(err).ToNot(HaveOccurred())
		ocpVersionMsg := fmt.Sprintf("K8s Version: %v\nOCP Version: %v\n", serverVersion.Kubernetes, serverVersion.Openshift)
		err = testutils.SaveToArtifactsDir([]byte(ocpVersionMsg), "OCP_Version.txt")
		Expect(err).ToNot(HaveOccurred())
		testutils.Printf("Info", ocpVersionMsg)
	})
})
