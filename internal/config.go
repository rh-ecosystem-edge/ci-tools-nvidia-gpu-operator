package internal

import (
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type config struct {
	NameSpace                string
	GpuOperatorChannel       string
	KubeconfigPath           string
	ArtifactDir              string
	CiMachineSetInstanceType string
	CiMachineSetReplicas     string
	ClientConfig             *rest.Config
}

var Config = config{
	NameSpace:                GetVarDefault("WORKING_NAMESPACE", "nvidia-gpu-operator"),
	GpuOperatorChannel:       GetVarDefault("GPU_CHANNEL", ""),
	KubeconfigPath:           GetVarDefault("KUBECONFIG", ".kubeconfig"),
	ArtifactDir:              GetVarDefault("ARTIFACT_DIR", "/tmp/gpu-test"),
	CiMachineSetInstanceType: GetVarDefault("GPU_INSTANCE_TYPE", "g4dn.xlarge"),
	CiMachineSetReplicas:     GetVarDefault("GPU_REPLICAS", "1"),
	ClientConfig:             GetClientConfig(),
}

func GetVarDefault(evar string, _default string) string {
	val := os.Getenv(evar)
	if len(val) == 0 {
		return _default
	}
	return val
}

func GetClientConfig() *rest.Config {
	kubeconfig := GetVarDefault("KUBECONFIG", ".kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err == nil {
		return config
	}
	config, err = rest.InClusterConfig()
	if err != nil {
		msg := fmt.Sprintf("Unable to create client config. invalid KUBECONFIG and %s", err)
		panic(msg)
	}
	return config
}
