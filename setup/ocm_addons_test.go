package setup

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

const (
	osde2eSecretNamespace = "osde2e-ci-secrets"
	osde2eSecretName      = "ci-secrets"
	gpuAddonId            = "nvidia-gpu-addon"
	rhodsAddonId          = "managed-odh"
)

type ocmAddonResponse struct {
	Kind            string `json:"kind"`
	State           string `json:"state,omitempty"`
	Id              string `json:"id,omitempty"`
	Code            string `json:"code,omitempty"`
	Reason          string `json:"reason,omitempty"`
	OperatorVersion string `json:"operator_version,omitempty"`
}
type ocmAddonPayload struct {
	Addon      ocmAddonPayloadAddon        `json:"addon"`
	Parameters []ocmAddonPayloadParameters `json:"parameters,omitempty"`
}
type ocmAddonPayloadAddon struct {
	Id string `json:"id"`
}
type ocmAddonPayloadParameters struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

var _ = Describe("ocm_addons_setup :", Ordered, func() {
	var (
		config       *rest.Config
		ocmToken     *string
		ocmEnv       *string
		ocmClusterId *string
	)
	BeforeAll(func() {

		config = internal.GetClientConfig()

		osde2eSecret, err := ocputils.GetSecret(config, osde2eSecretNamespace, osde2eSecretName)
		Expect(err).ToNot(HaveOccurred())
		Expect(osde2eSecret).ToNot(BeNil())

		ocmToken, err = ocputils.GetSecretValue(osde2eSecret, "ocm-token-refresh", false)
		Expect(err).ToNot(HaveOccurred())
		ocmEnv, err = ocputils.GetSecretValue(osde2eSecret, "ENV", false)
		Expect(err).ToNot(HaveOccurred())
		ocmClusterId, err = ocputils.GetSecretValue(osde2eSecret, "CLUSTER_ID", false)
		Expect(err).ToNot(HaveOccurred())

		Expect(ocmToken).NotTo(BeNil())
		Expect(ocmEnv).NotTo(BeNil())
		Expect(ocmClusterId).NotTo(BeNil())
	})
	It("login to ocm successfully", func() {
		cmd := fmt.Sprintf("login --token=%s --url=%s", *ocmToken, *ocmEnv)
		_, err := runOcmCommand(cmd)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("RHODS installation", Ordered, func() {
		var addonInstalled bool

		BeforeAll(func() {
			if *ocmEnv != "prod" {
				Skip("No need to install RHODS on environments other than production")
				return
			}
		})

		It("get RHODS install state", func() {
			resp := getAddon(*ocmClusterId, rhodsAddonId)
			Expect(resp.Kind).ToNot(BeEmpty())
			if resp.Kind == "AddOnInstallation" {
				testutils.Printf("info", "RHODS seems to already be installed")
				addonInstalled = true
			} else {
				testutils.Printf("info", "RHODS Addon not found in cluster.")
			}
			err := testutils.SaveAsJsonToArtifactsDir(resp, "rhods-addon-initial-response.json")
			Expect(err).ToNot(HaveOccurred())
		})
		Context("install addon if needed", func() {
			var (
				payloadFileName = "ocm-rhods-addon-payload.json"
				payloadPath     = fmt.Sprintf("%s/%s", internal.Config.ArtifactDir, payloadFileName)
			)
			if addonInstalled {
				Skip("RHODS addon is already installed on cluster")
				return
			}

			It("prepare addon payload", func() {
				payload := &ocmAddonPayload{}
				payload.Addon.Id = gpuAddonId
				payload.Parameters = []ocmAddonPayloadParameters{
					{
						Id:    "notification-email",
						Value: "example@example.com",
					},
				}

				err := testutils.SaveAsJsonToArtifactsDir(payload, payloadFileName)
				Expect(err).ToNot(HaveOccurred())
			})

			It("install RHODS addon", func() {
				url := fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/addons", *ocmClusterId)
				cmd := fmt.Sprintf("post %s --body=%s", url, payloadPath)
				out, err := runOcmCommand(cmd)
				Expect(err).ToNot(HaveOccurred())
				err = testutils.SaveToArtifactsDir([]byte(out), "ocm-addon-install-resp.json")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("wait for rhods addon to be installed", func() {
			installed, err := waitForAddonToBeInstalled(*ocmClusterId, rhodsAddonId)
			Expect(err).ToNot(HaveOccurred())
			Expect(installed).To(BeTrue(), "addon was not installed correctly")
		})
	})

	Context("GPU addon installation", Ordered, func() {
		var (
			addonInstalled bool
		)
		It("get state of gpu-addon installation", func() {
			resp := getAddon(*ocmClusterId, gpuAddonId)
			Expect(resp.Kind).ToNot(BeEmpty())
			if resp.Kind == "AddOnInstallation" {
				testutils.Printf("info", "GPU Addon seems to already be installed")
				addonInstalled = true
			} else {
				testutils.Printf("info", "GPU Addon not found in cluster.")
			}
			err := testutils.SaveAsJsonToArtifactsDir(resp, "gpu-addon-initial-response.json")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("install addon if needed", Ordered, func() {
			var (
				payloadFileName = "ocm-gpu-addon-payload.json"
				payloadPath     = fmt.Sprintf("%s/%s", internal.Config.ArtifactDir, payloadFileName)
			)
			BeforeAll(func() {
				if addonInstalled {
					Skip("GPU addon is already installed on cluster")
					return
				}
			})

			It("prepare addon payload", func() {
				payload := &ocmAddonPayload{}
				payload.Addon.Id = gpuAddonId

				err := testutils.SaveAsJsonToArtifactsDir(payload, payloadFileName)
				Expect(err).ToNot(HaveOccurred())
			})

			It("install gpu addon", func() {
				url := fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/addons", *ocmClusterId)
				cmd := fmt.Sprintf("post %s --body=%s", url, payloadPath)
				out, err := runOcmCommand(cmd)
				Expect(err).ToNot(HaveOccurred())
				err = testutils.SaveToArtifactsDir([]byte(out), "ocm-addon-install-resp.json")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("wait for gpu addon to be installed", func() {
			installed, err := waitForAddonToBeInstalled(*ocmClusterId, gpuAddonId)
			Expect(err).ToNot(HaveOccurred())
			Expect(installed).To(BeTrue(), "addon was not installed correctly")
		})
	})

})

func getAddon(clusterId string, addonId string) *ocmAddonResponse {
	cmd := fmt.Sprintf("get /api/clusters_mgmt/v1/clusters/%s/addons/%s", clusterId, addonId)
	out, err := runOcmCommand(cmd)
	resp := &ocmAddonResponse{}
	if err != nil {
		_ = json.Unmarshal([]byte(err.Error()), resp)
	} else {
		_ = json.Unmarshal([]byte(out), resp)
	}
	return resp
}

func waitForAddonToBeInstalled(clusterId string, addonId string) (bool, error) {
	var installed bool
	filename := fmt.Sprintf("addon-%v-ocm-response.json", addonId)
	apiResp := &ocmAddonResponse{}
	err := testutils.ExecWithRetryBackoff("wait for gpu-addon install state", func() bool {
		apiResp = getAddon(clusterId, addonId)
		if len(apiResp.Kind) == 0 {
			return false
		} else if apiResp.Kind == "Error" {
			return false
		}
		switch apiResp.State {
		case "ready":
			installed = true
			return true
		case "failed", "deleting":
			// error - stop the retries
			return true
		}
		return false
	}, 60, 60*time.Second)
	_ = testutils.SaveAsJsonToArtifactsDir(apiResp, filename)
	return installed, err
}

func runOcmCommand(command string) (string, error) {
	cmd := exec.Command("ocm", strings.Split(command, " ")...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", out)
	}
	return string(out), nil
}
