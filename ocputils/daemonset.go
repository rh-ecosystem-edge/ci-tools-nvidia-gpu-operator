package ocputils

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetDaemonset(config *rest.Config, namespace string, name string) (*appsv1.DaemonSet, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func CreatDaemonSet(config *rest.Config, ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.AppsV1().DaemonSets(ds.Namespace).Create(context.TODO(), ds, metav1.CreateOptions{})
}
