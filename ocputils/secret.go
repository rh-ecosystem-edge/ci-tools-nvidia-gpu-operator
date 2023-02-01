package ocputils

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetSecret(config *rest.Config, namespace string, name string) (*corev1.Secret, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func GetSecretValue(secret *corev1.Secret, data string, isGziped bool) (*string, error) {
	val, ok := secret.Data[data]
	if !ok {
		return nil, fmt.Errorf("Data %v not found in secret %v", data, secret.Name)
	}
	if isGziped {
		reader := bytes.NewReader(val)
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		val, err = io.ReadAll(gzReader)
		if err != nil {
			return nil, err
		}
	}
	returnStr := string(val)
	return &returnStr, nil
}
