package ocputils

import (
	"bytes"
	"context"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetPodsByLabel(config *rest.Config, namespace string, labelSelector string) (*corev1.PodList, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

func GetPodLogs(config *rest.Config, pod corev1.Pod, follow bool) (*string, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow: follow,
	})
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, err
	}
	str := buf.String()
	str = strings.ReplaceAll(str, "\r", "\n")
	return &str, nil
}

func PodProxyGet(config *rest.Config, pod corev1.Pod, port string, path string, params map[string]string) ([]byte, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	foo := clientset.CoreV1().Pods(pod.Namespace).ProxyGet("", pod.Name, port, path, params)
	resp, err := foo.DoRaw(context.TODO())
	if err != nil {
		return nil, err
	}
	return resp, nil
}
