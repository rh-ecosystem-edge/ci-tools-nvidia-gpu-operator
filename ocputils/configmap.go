package ocputils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CreateConfigMap(config *rest.Config, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().ConfigMaps(cm.Namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
}
