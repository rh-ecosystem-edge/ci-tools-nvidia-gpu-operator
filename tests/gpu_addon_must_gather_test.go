package tests

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/client-go/rest"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("gpu_addon_must_gather :", Ordered, func() {
	var (
		config          *rest.Config
		gpuAddonCsv     *operatorsv1alpha1.ClusterServiceVersion
		mustGatherImage string
	)

	BeforeAll(func() {
		config = internal.GetClientConfig()

	})
	It("fetch must gather image from csv", func() {
		csvs, err := ocputils.GetCsvsByLabel(config, "", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(csvs.Items).ToNot(BeEmpty())
		for _, csv := range csvs.Items {
			if strings.Contains(csv.Name, "nvidia-gpu-addon") {
				gpuAddonCsv = &csv
				break
			}
		}
		Expect(gpuAddonCsv).ToNot(BeNil(), "CSV not found")
		err = testutils.SaveAsJsonToArtifactsDir(gpuAddonCsv, "gpu-addon-csv.json")
		Expect(err).ToNot(HaveOccurred())
		testutils.Printf("Info", "GPU add-on name=%v namespace=%v version=%v", gpuAddonCsv.Name, gpuAddonCsv.Namespace, gpuAddonCsv.Spec.Version.String())
		for _, image := range gpuAddonCsv.Spec.RelatedImages {
			if image.Name == "must-gather" {
				mustGatherImage = image.Image
				break
			}
		}
		Expect(mustGatherImage).ToNot(BeEmpty(), "must-gather image not found")
		testutils.Printf("Info", "must-gather image: %s", mustGatherImage)
	})

	It("run addon must-gather", func() {
		Expect(mustGatherImage).ToNot(BeEmpty(), "must-gather image not found")
		cmd := fmt.Sprintf("adm must-gather --image=%s --dest-dir=%s", mustGatherImage, internal.Config.ArtifactDir)
		out, err := runOcCommand(cmd)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveToArtifactsDir([]byte(out), "oc-must-gather-output.txt")
		Expect(err).ToNot(HaveOccurred())
	})
})

func runOcCommand(command string) (string, error) {
	cmd := exec.Command("oc", strings.Split(command, " ")...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", out)
	}
	return string(out), nil
}
