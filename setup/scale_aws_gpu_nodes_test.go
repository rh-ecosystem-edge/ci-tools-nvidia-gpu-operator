package setup

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	machinesetv1b1 "github.com/openshift/api/machine/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("scale_aws_gpu_nodes : ", Ordered, func() {

	var (
		config        *rest.Config
		gpuMachineset *machinesetv1b1.MachineSet
		kubeconfig    string
		instanceType  string
		namespace     string
		replicas      int32
	)

	BeforeAll(func() {
		kubeconfig = internal.Config.KubeconfigPath
		instanceType = internal.Config.CiMachineSetInstanceType
		namespace = "openshift-machine-api"

		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		Expect(err).ToNot(HaveOccurred())
		r, err := strconv.ParseInt(internal.Config.CiMachineSetReplicas, 10, 32)
		Expect(err).ToNot(HaveOccurred(), "Invalid replicas value")
		replicas = int32(r)
	})

	It("ensure a Machineset for desired instance type", func() {
		workerMs, err := ocputils.GetWorkerMachineSets(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerMs.Items).ToNot(BeEmpty(), "No worker Machinesets found")
		for _, ms := range workerMs.Items {
			fileName := fmt.Sprintf("worker_ms-%v.json", ms.Name)
			_ = testutils.SaveAsJsonToArtifactsDir(ms, fileName)
			m, err := mapFromProviderSpec(ms)
			Expect(err).ToNot(HaveOccurred())
			if val, ok := m["instanceType"]; ok && val == instanceType {
				gpuMachineset = &ms
				break
			}
		}
	})

	It("create a Machineset for desired instance type", func() {
		if gpuMachineset != nil {
			Skip("Machineset for desired instance exists. Skipping")
		}

		workerMs, err := ocputils.GetWorkerMachineSets(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerMs.Items).ToNot(BeEmpty(), "No worker Machinesets found")

		baseMs := workerMs.Items[0]

		ms := &machinesetv1b1.MachineSet{
			ObjectMeta: *baseMs.ObjectMeta.DeepCopy(),
			Spec:       *baseMs.Spec.DeepCopy(),
		}
		testutils.Printf("Info", "Using Machineset %v as base for new machineset", ms.Name)
		// Change meta
		ms.ObjectMeta.Name = fmt.Sprintf("%v-%v", ms.Name, strings.ReplaceAll(instanceType, ".", "-"))
		ms.ObjectMeta.UID = ""
		ms.ObjectMeta.ResourceVersion = ""
		// chenge spec labels
		ms.Spec.Selector.MatchLabels["machine.openshift.io/cluster-api-machineset"] = ms.ObjectMeta.Name
		ms.Spec.Template.ObjectMeta.Labels["machine.openshift.io/cluster-api-machineset"] = ms.ObjectMeta.Name
		// Change instance type
		err = changeProviderInstanceType(ms, instanceType)
		Expect(err).ToNot(HaveOccurred())
		// Set replicas to 1
		ms.Spec.Replicas = &replicas

		ms, err = ocputils.CreateMachineSet(config, namespace, ms)
		Expect(err).ToNot(HaveOccurred())

		gpuMachineset = ms

		fileName := fmt.Sprintf("new_machineset-%v.json", ms.Name)
		err = testutils.SaveAsJsonToArtifactsDir(ms, fileName)
		Expect(err).ToNot(HaveOccurred())
	})

	It("ensure number of replicas on MachineSet", func() {
		if gpuMachineset.Spec.Replicas != &replicas {
			patch := fmt.Sprintf("{\"spec\": {\"replicas\": %v}}", replicas)
			ms, err := ocputils.PatchMachineSet(config, gpuMachineset, []byte(patch), types.MergePatchType)
			Expect(err).ToNot(HaveOccurred())
			gpuMachineset = ms
		}
	})

	It("wait for GPU MachineSet to become ready", func() {
		err := testutils.ExecWithRetryBackoff("MachineSet ready state", func() bool {
			if gpuMachineset.Status.ReadyReplicas != *gpuMachineset.Spec.Replicas {
				ms, err := ocputils.GetMachineSet(config, namespace, gpuMachineset.Name)
				if err == nil {
					gpuMachineset = ms
				}
				return false
			}
			return true
		}, 30, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
	})
})

func mapFromProviderSpec(ms machinesetv1b1.MachineSet) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	b, err := json.Marshal(ms.Spec.Template.Spec.ProviderSpec.Value)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func changeProviderInstanceType(ms *machinesetv1b1.MachineSet, instanceType string) error {
	providerSpecMap, err := mapFromProviderSpec(*ms)
	if err != nil {
		return err
	}
	providerSpecMap["instanceType"] = instanceType
	b, err := json.Marshal(providerSpecMap)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, ms.Spec.Template.Spec.ProviderSpec.Value)
	if err != nil {
		return err
	}
	return nil
}
