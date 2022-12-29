package setup

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("deploy_nfd_operator :", Ordered, func() {
	var (
		config              *rest.Config
		kubeconfig          string
		nfdOpName           string
		nfdChannel          string
		nfdCatalogSource    string
		nfdCatalogSourceNS  string
		nfdCsvLabelSelector string
		nfdPkgNS            string
	)

	BeforeAll(func() {
		kubeconfig = internal.Config.KubeconfigPath
		nfdOpName = "nfd"
		nfdChannel = "unset"
		nfdCatalogSource = "unset"
		nfdCatalogSourceNS = "unset"
		nfdCsvLabelSelector = "unset"
		nfdPkgNS = "openshift-marketplace"

		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		Expect(err).ToNot(HaveOccurred())
	})

	It("check NFD PackageManifest", func() {
		pkg, err := ocputils.GetPackageManifest(config, nfdPkgNS, "nfd")
		Expect(err).ToNot(HaveOccurred())
		nfdChannel = pkg.Status.DefaultChannel
		nfdCatalogSource = pkg.Status.CatalogSource
		nfdCatalogSourceNS = pkg.Status.CatalogSourceNamespace
		testutils.Printf("PKG Manifest", "NFD Operator PackageManifest defaultchannel '%v'", nfdChannel)
		err = testutils.SaveAsJsonToArtifactsDir(pkg, "nfd_packagemanifest.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("ensure namespace exists", func() {
		ns, err := ocputils.CreateNamespace(config, internal.Config.NameSpace)
		if !errors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(ns).ToNot(BeNil())
	})

	It("create Operator Group", func() {
		_, err := ocputils.CreateOperatorGroup(config, internal.Config.NameSpace, "ci-group")
		if !errors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}

	})

	It("create Subscription", func() {
		sub, err := ocputils.CreateSubscription(config, internal.Config.NameSpace, "nfd", nfdChannel, "nfd", nfdCatalogSource, nfdCatalogSourceNS)
		if !errors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		err = testutils.SaveAsJsonToArtifactsDir(sub, "nfd_subscription.json")
		Expect(err).ToNot(HaveOccurred())
		nfdCsvLabelSelector = fmt.Sprintf("operators.coreos.com/%v.%v", nfdOpName, internal.Config.NameSpace)
	})

	It("wait Until CSV is installed", func() {
		csv, err := waitForCsvPhase(config, internal.Config.NameSpace, nfdCsvLabelSelector, "Succeeded")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(csv, "nfd_csv.json")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveToArtifactsDir([]byte(csv.Spec.Version.String()), "nfd_version.txt")
		Expect(err).ToNot(HaveOccurred())
	})

	It("deploy NFD CR", func() {
		nfd := &nfdv1.NodeFeatureDiscovery{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeFeatureDiscovery",
				APIVersion: nfdv1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: internal.Config.NameSpace,
				Name:      "nfd-cr-testing",
			},
		}

		resp, err := ocputils.CreateDynamicResource(config, nfdv1.GroupVersion.WithResource("nodefeaturediscoveries"), nfd, nfd.Namespace)
		Expect(err).ToNot(HaveOccurred())
		var respNfd nfdv1.NodeFeatureDiscovery = nfdv1.NodeFeatureDiscovery{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resp.UnstructuredContent(), &respNfd)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(respNfd, "nfd_cr.json")
		Expect(err).ToNot(HaveOccurred())

	})

	It("wait for NFD labels", func() {
		err := testutils.ExecWithRetryBackoff("wait for NFD labels", func() bool {
			nodes, err := ocputils.GetNodesByLabel(config, "feature.node.kubernetes.io/system-os_release.ID=rhcos")
			if err != nil {
				return false
			}
			return len(nodes.Items) > 0
		}, 20, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
	})
})
