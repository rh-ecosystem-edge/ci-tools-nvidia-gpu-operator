package setup

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

const (
	nfdCrName   = "nfd-cr-testing"
	nfdResource = "nodefeaturediscoveries"
)

var _ = Describe("deploy_nfd_operator :", Ordered, func() {
	var (
		config              *rest.Config
		nfdOpName           string
		nfdChannel          string
		nfdCatalogSource    string
		nfdCatalogSourceNS  string
		nfdCsvLabelSelector string
		nfdPkgNS            string
		nfdAlmExample       string
	)

	BeforeAll(func() {
		nfdOpName = "nfd"
		nfdChannel = "unset"
		nfdCatalogSource = "unset"
		nfdCatalogSourceNS = "unset"
		nfdCsvLabelSelector = "unset"
		nfdPkgNS = "openshift-marketplace"

		config = internal.GetClientConfig()
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
		_ = testutils.SaveAsJsonToArtifactsDir(ns, "namespace.json")
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

	It("wait until CSV is installed and capture alm example", func() {
		csv, err := waitForCsvPhase(config, internal.Config.NameSpace, nfdCsvLabelSelector, "Succeeded")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(csv, "nfd_csv.json")
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveToArtifactsDir([]byte(csv.Spec.Version.String()), "nfd_version.txt")
		Expect(err).ToNot(HaveOccurred())
		nfdAlmExample, err = ocputils.GetAlmExamples(&csv)
		Expect(err).ToNot(HaveOccurred())
		Expect(nfdAlmExample).ToNot(BeEmpty())
	})

	It("deploy NFD CR based on alm example", func() {
		unstructObj, err := getUnstructuredFromAlmExample(nfdAlmExample)
		Expect(err).ToNot(HaveOccurred())
		unstructObj.SetNamespace(internal.Config.NameSpace)
		unstructObj.SetName(nfdCrName)
		resp, err := ocputils.CreateDynamicResource(config, nfdv1.GroupVersion.WithResource(nfdResource), unstructObj, internal.Config.NameSpace)
		Expect(err).ToNot(HaveOccurred())
		var respNfd nfdv1.NodeFeatureDiscovery = nfdv1.NodeFeatureDiscovery{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resp.UnstructuredContent(), &respNfd)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(respNfd, "nfd_cr_create_response.json")
		Expect(err).ToNot(HaveOccurred())

	})

	It("wait for NFD labels and capture nfd cr state", func() {
		err := testutils.ExecWithRetryBackoff("wait for NFD labels", func() bool {
			nodes, err := ocputils.GetNodesByLabel(config, "feature.node.kubernetes.io/system-os_release.ID=rhcos")
			if err != nil {
				return false
			}
			return len(nodes.Items) > 0
		}, 20, 30*time.Second)

		defer Expect(err).ToNot(HaveOccurred())
		if err != nil {
			testutils.Printf("error", "Failed to find labels on nodes. %v", err)
		}

		// Regarless if successful, we want to have the CR in artifacts
		nfdCr := &nfdv1.NodeFeatureDiscovery{}
		e := ocputils.GetDynamicResource(config, nfdv1.GroupVersion.WithResource(nfdResource), internal.Config.NameSpace, nfdCrName, nfdCr)
		Expect(e).ToNot(HaveOccurred())
		e = testutils.SaveAsJsonToArtifactsDir(nfdCr, "nfd_cr.json")
		Expect(e).ToNot(HaveOccurred())
	})
})
