package ocputils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CreateNamespace(config *rest.Config, name string) (*corev1.Namespace, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
}

func DeleteNamespace(config *rest.Config, name string) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	return clientset.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
}
func GetNamespace(config *rest.Config, name string) (*corev1.Namespace, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}
