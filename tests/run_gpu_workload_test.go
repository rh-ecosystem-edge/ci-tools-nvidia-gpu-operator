package tests

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

var _ = Describe("run_gpu_workload :", Ordered, func() {
	var (
		config        *rest.Config
		namespace     string
		gpuBurnImage  string
		daemonsetName string
	)

	BeforeAll(func() {
		namespace = "gpu-burn-test"
		gpuBurnImage = "quay.io/openshift-psap/gpu-burn"
		daemonsetName = "gpu-burn-daemonset"

		config = internal.GetClientConfig()
	})

	It("create gpu-burn namespace", func() {
		ns, err := ocputils.CreateNamespace(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(ns, "gpu_burn_namespace.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("create gpu-burn ConfigNap", func() {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gpu-burn-entrypoint",
				Namespace: namespace,
			},
			Data: map[string]string{
				"entrypoint.sh": `#!/bin/bash
                                  NUM_GPUS=$(nvidia-smi -L | wc -l)
                                  if [ $NUM_GPUS -eq 0 ]; then
                                    echo "ERROR No GPUs found"
                                    exit 1
                                  fi
                                  /usr/local/bin/gpu-burn 300
                                  if [ ! $? -eq 0 ]; then 
                                    exit 1
                                  fi`,
			},
		}
		cm, err := ocputils.CreateConfigMap(config, cm)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(cm, "gpu_burn_configmap.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("create gpu-burn DaemonSet", func() {
		ds := newBurnDaemonSet(namespace, daemonsetName, gpuBurnImage)
		ds, err := ocputils.CreatDaemonSet(config, ds)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.SaveAsJsonToArtifactsDir(ds, "gpu_burn_daemonset.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("daemon set should run", func() {
		var ds *appsv1.DaemonSet
		err := testutils.ExecWithRetryBackoff("DaemonSet state check. Desired vs Ready", func() bool {
			var err error
			ds, err = ocputils.GetDaemonset(config, namespace, daemonsetName)
			if err != nil {
				return false
			}
			return ds.Status.NumberReady == ds.Status.DesiredNumberScheduled
		}, 20, 30*time.Second)
		Expect(err).ToNot(HaveOccurred(), "Desired != Ready")
		err = testutils.SaveAsJsonToArtifactsDir(ds, "gpu_burn_daemonset.json")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should run burn to completion on all nodes", func() {
		var pods *corev1.PodList
		err := testutils.ExecWithRetryBackoff("Get Daemonset pods", func() bool {
			var err error
			pods, err = ocputils.GetPodsByLabel(config, namespace, "app=gpu-burn-daemonset")
			if err != nil {
				return false
			}
			if len(pods.Items) == 0 {
				return false // Pods not ready yet
			}
			return true
		}, 15, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())
		podStates := map[string]bool{}
		err = testutils.ExecWithRetryBackoff("Wait for GPU Burn to finish", func() bool {
			for _, pod := range pods.Items {
				if val, ok := podStates[pod.Name]; ok && val {
					continue
				}
				output_resp, err := ocputils.GetPodLogs(config, pod, true)
				if err != nil {
					return false
				}
				output := *output_resp
				filename := fmt.Sprintf("pod_%v_output.log", pod.Name)
				_ = testutils.SaveToArtifactsDir([]byte(output), filename)
				match1 := strings.Contains(output, "GPU 0: OK")
				match2 := strings.Contains(output, "100.0%  proc'd:")
				podStates[pod.Name] = match1 && match2
			}
			if len(podStates) != len(pods.Items) {
				return false
			}
			for _, ok := range podStates {
				if !ok {
					return false
				}
			}
			return true

		}, 60, 1*time.Minute)
		Expect(err).ToNot(HaveOccurred())
	})

	It("successfully remove bun test namespace", func() {
		err := ocputils.DeleteNamespace(config, namespace)
		Expect(err).ToNot(HaveOccurred())
		err = testutils.ExecWithRetryBackoff("Wait until namespace is deleted", func() bool {
			_, err := ocputils.GetNamespace(config, namespace)
			return errors.IsNotFound(err)
		}, 60, 10*time.Second)
		Expect(err).ToNot(HaveOccurred())
	})
})
