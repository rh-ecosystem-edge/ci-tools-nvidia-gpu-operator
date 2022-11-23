package setup

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	pkgmanifestv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

const (
	deployMasterEnvVar = "DEPLOY_GPU_OP_NVIDIA_MAIN"
)

var _ = Describe("deploy_gpu_operator :", Ordered, func() {
	kubeconfig := internal.Config.KubeconfigPath
	var (
		config                *rest.Config
		pkg                   *pkgmanifestv1.PackageManifest
		catalogSource         string
		gpuOpChannel          string
		catalogSourceNS       string
		operatorPkgName       string
		clusterServiceVersion *operatorsv1alpha1.ClusterServiceVersion
	)
	BeforeAll(func() {
		catalogSource = "certified-operators"
		gpuOpChannel = internal.Config.GpuOperatorChannel
		catalogSourceNS = "openshift-marketplace"
		operatorPkgName = "gpu-operator-certified"

		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		Expect(err).ToNot(HaveOccurred())
	})
	Context("from certified operators", Ordered, func() {
		BeforeAll(func() {
			if testutils.SkipTestIfEnvVarSet(deployMasterEnvVar, true) {
				return
			}
		})

		It("make shure certified operators catalog is ready", func() {
			cSourse, err := ocputils.GetCatalogSource(config, catalogSourceNS, catalogSource)
			Expect(err).ToNot(HaveOccurred())
			Expect(cSourse.Status.GRPCConnectionState.LastObservedState).To(BeEquivalentTo("READY"))
			testutils.Printf("certified-operators CatalogSource state:", "%v", cSourse.Status.GRPCConnectionState.LastObservedState)
		})

		It("check if GPU Operator is in catalog", func() {
			var err error
			pkg, err = ocputils.GetPackageManifest(config, catalogSourceNS, operatorPkgName)
			Expect(err).ToNot(HaveOccurred())
			testutils.Printf("PKG Manifest", "GPU Operator PackageManifest current version [%v]: %v", pkg.Status.Channels[0].Name, pkg.Status.Channels[0].CurrentCSV)
			if len(gpuOpChannel) == 0 {
				// No channel specified - use latest
				gpuOpChannel = pkg.Status.Channels[0].Name
			}
			err = testutils.SaveAsJsonToArtifactsDir(pkg, "gpu_operator_packagemanifest.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("make sure channel exists in packagemanifest", func() {
			testutils.Printf("GPU operator channel", "Channel=%v", gpuOpChannel)
			var found bool
			for _, channel := range pkg.Status.Channels {
				if channel.Name == gpuOpChannel {
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})

		It("deploy GPU operator", func() {
			subName := "gpu-operator-test-sub"
			sub, err := ocputils.CreateSubscription(config, internal.Config.NameSpace, subName,
				gpuOpChannel, operatorPkgName, catalogSource, catalogSourceNS)
			Expect(err).ToNot(HaveOccurred())
			err = testutils.SaveAsJsonToArtifactsDir(sub, "gpu_operator_subscription.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("wait Until CSV is installed", func() {
			labelSelector := fmt.Sprintf("operators.coreos.com/%v.%v", operatorPkgName, internal.Config.NameSpace)
			csv, err := waitForCsvPhase(config, internal.Config.NameSpace, labelSelector, "Succeeded")
			Expect(err).ToNot(HaveOccurred())
			err = testutils.SaveAsJsonToArtifactsDir(csv, "gpu_operator_csv.json")
			Expect(err).ToNot(HaveOccurred())
			err = testutils.SaveToArtifactsDir([]byte(csv.Spec.Version.String()), "gpu_operator_version.txt")
			Expect(err).ToNot(HaveOccurred())
			clusterServiceVersion = &csv
		})

		It("deploy GPU ClusterPolicy", func() {
			almExample, err := ocputils.GetAlmExamples(clusterServiceVersion)
			Expect(err).ToNot(HaveOccurred())
			cpJson := unstructured.UnstructuredList{}
			err = json.Unmarshal([]byte(almExample), &cpJson.Items)
			Expect(err).ToNot(HaveOccurred())
			Expect(cpJson.Items).ToNot(BeEmpty())
			unstructObj := cpJson.Items[0]
			unstructObj.SetNamespace(internal.Config.NameSpace)

			resp, err := ocputils.CreateDynamicResource(config, gpuv1.GroupVersion.WithResource("clusterpolicies"), &unstructObj, "")

			Expect(err).ToNot(HaveOccurred())
			respCp := gpuv1.ClusterPolicy{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(resp.UnstructuredContent(), &respCp)
			Expect(err).ToNot(HaveOccurred())
			err = testutils.SaveAsJsonToArtifactsDir(respCp, "gpu_cr_cluster_policy.json")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("NVIDIA's Main branch Bundle", Ordered, func() {
		BeforeAll(func() {
			if testutils.SkipTestIfEnvVarSet(deployMasterEnvVar, false) {
				return
			}
		})
		It("deploy Prebuilt nvidia bundle.", func() {

		})
	})

})
