package tests

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	no  bool = false
	yes bool = true
)

func newBurnDaemonSet(namespace string, name string, gpuBurnImage string) *appsv1.DaemonSet {
	var volumeDefaultMode int32 = 0777
	configMapVolumeSource := &corev1.ConfigMapVolumeSource{}
	configMapVolumeSource.Name = "gpu-burn-entrypoint"
	configMapVolumeSource.DefaultMode = &volumeDefaultMode
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "gpu-burn-daemonset",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "gpu-burn-daemonset",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "gpu-burn-daemonset",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &yes,
					},
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
						},
						{
							Key:      "nvidia.com/gpu",
							Effect:   corev1.TaintEffectNoSchedule,
							Operator: corev1.TolerationOpExists,
						},
					},
					Containers: []corev1.Container{
						{
							Image:           gpuBurnImage,
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &no,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
							},
							Name: "gpu-burn-ctr",
							Command: []string{
								"/bin/entrypoint.sh",
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"nvidia.com/gpu": resource.MustParse("1"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "entrypoint",
									MountPath: "/bin/entrypoint.sh",
									ReadOnly:  true,
									SubPath:   "entrypoint.sh",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "entrypoint",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: configMapVolumeSource,
							},
						},
					},
					NodeSelector: map[string]string{
						"nvidia.com/gpu.present":         "true",
						"node-role.kubernetes.io/worker": "",
					},
				},
			},
		},
	}

}
