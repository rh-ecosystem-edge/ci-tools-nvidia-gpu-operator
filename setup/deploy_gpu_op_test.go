package setup

import (
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	pkgmanifestv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

const (
	deployedFromMaster = "DEPLOYED_FROM_MASTER"
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

	It("ensure namespace exists", func() {
		ns, err := ocputils.CreateNamespace(config, internal.Config.NameSpace)
		if !errors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(ns).ToNot(BeNil())
		_ = testutils.SaveAsJsonToArtifactsDir(ns, "namespace.json")
	})

	Context("from certified operators", Ordered, func() {
		BeforeAll(func() {
			if testutils.SkipTestIfEnvVarSet(deployedFromMaster, true) {
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
			testutils.Printf("PKG Manifest", "GPU Operator PackageManifest default channel '%v'", pkg.Status.DefaultChannel)
			if len(gpuOpChannel) == 0 {
				// No channel specified - use defaultChannel
				gpuOpChannel = pkg.Status.DefaultChannel
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
	})

	Context("post install setup", Ordered, func() {
		var (
			namespace string
		)

		It("get csv", func() {
			err := testutils.ExecWithRetryBackoff("get gpu op csv", func() bool {
				csvs, err := ocputils.GetCsvsByLabel(config, "", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(csvs.Items).ToNot(BeEmpty())
				for _, csv := range csvs.Items {
					if strings.Contains(csv.Name, "gpu-operator-certified") {
						clusterServiceVersion = &csv
						namespace = csv.Namespace
						return true
					}
				}
				return false
			}, 30, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(clusterServiceVersion).ToNot(BeNil(), "CSV not found")
			Expect(namespace).ToNot(BeEmpty())
		})
		It("wait Until CSV is installed", func() {
			succeeded := operatorsv1alpha1.ClusterServiceVersionPhase("Succeeded")
			testutils.Printf("Info", "GPU Operator name=%v namespace=%v version=%v", clusterServiceVersion.Name, clusterServiceVersion.Namespace, clusterServiceVersion.Spec.Version.String())
			if clusterServiceVersion.Status.Phase != succeeded {
				err := testutils.ExecWithRetryBackoff("Wait for CSV to be Succeeded", func() bool {
					csv, err := ocputils.GetCsvByName(config, clusterServiceVersion.Namespace, clusterServiceVersion.Name)
					if err != nil {
						return false
					}
					clusterServiceVersion = csv
					return clusterServiceVersion.Status.Phase == succeeded
				}, 15, 30*time.Second)
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(clusterServiceVersion.Status.Phase).To(Equal(succeeded), "CSV Phase is not Succeeded")
			err := testutils.SaveAsJsonToArtifactsDir(clusterServiceVersion, "gpu_operator_csv.json")
			Expect(err).ToNot(HaveOccurred())
			err = testutils.SaveToArtifactsDir([]byte(clusterServiceVersion.Spec.Version.String()), "gpu_operator_version.txt")
			Expect(err).ToNot(HaveOccurred())
		})

		It("deploy GPU ClusterPolicy", func() {
			almExample, err := ocputils.GetAlmExamples(clusterServiceVersion)
			Expect(err).ToNot(HaveOccurred())
			unstructObj, err := getUnstructuredFromAlmExample(almExample)
			Expect(err).ToNot(HaveOccurred())
			unstructObj.SetNamespace(namespace)

			resp, err := ocputils.CreateDynamicResource(config, gpuv1.GroupVersion.WithResource("clusterpolicies"), unstructObj, "")

			Expect(err).ToNot(HaveOccurred())
			respCp := gpuv1.ClusterPolicy{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(resp.UnstructuredContent(), &respCp)
			Expect(err).ToNot(HaveOccurred())
			err = testutils.SaveAsJsonToArtifactsDir(respCp, "gpu_cr_cluster_policy.json")
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
