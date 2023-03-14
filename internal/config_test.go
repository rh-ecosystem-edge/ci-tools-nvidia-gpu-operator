package internal

import (
	"fmt"
	"os"
	"testing"
)

func Check(err error, msg string) bool {
	if err == nil {
		return true
	}
	panic(msg)
}

func CreateFakeKubeConfig() {
	data := []byte(`
	apiVersion: v1
clusters:
- cluster:
    certificate-authority: /fake/.minikube/ca.crt
    server: https://10.10.10.10:8443
  name: minikube
contexts:
- context:
    cluster: minikube
    user: minikube
  name: minikube
current-context: minikube
kind: Config
preferences: {}
users:
- name: minikube
  user:
    client-certificate-data: /workdir/.minikube/client.crt
    client-key-data: /workdir/.minikube/client.key
`)

	file, err := os.Create(".kubeconfig")
	Check(err, "Cannot create file .kubeconfig")

	defer file.Close()
	_, err = file.Write(data)
	Check(err, "Cannot Write file .kubeconfig")
	err = file.Sync()
	Check(err, "Failed to flush file data")

}

func TestGetVarDefault(t *testing.T) {
	// Test case when environment variable is set
	name, val := "TestGetVarDefault", "value"
	os.Setenv(name, val)
	actual := GetVarDefault(name, "default")
	if actual != val {
		t.Errorf("GetVarDefault returned wrong value for set environment variable, expected: %s, got: %s", val, actual)
	}

	//Test case when environment variable is not set
	val = "default"
	actual = GetVarDefault("UNDEFINED", val)
	if actual != val {
		t.Errorf("GetVarDefault returned wrong value for unset environment variable, expected: %s, got: %s", val, actual)
	}
	// Test case when environment variable is empty string
	name, val, expected := "TestGetVarDefault", "", "default"
	os.Setenv(name, val)
	actual = GetVarDefault(name, expected)
	if actual != expected {
		t.Errorf("GetVarDefault returned wrong value for empty string, expected: %s, got: %s", expected, actual)
	}
}

func TestGetClientConfig(t *testing.T) {
	// create fake kubeconfig file
	CreateFakeKubeConfig()

	config, expected := GetClientConfig(), "*rest.Config"
	if (fmt.Sprintf("%T", config)) != "*rest.Config" {
		t.Errorf("GetClientConfig returned wrong config, expected: %v, got: %T", expected, config)
	}
	os.Remove(".kubeconfig")
}
