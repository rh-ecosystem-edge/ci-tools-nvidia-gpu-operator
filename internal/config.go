package internal

import (
	"os"
)

type config struct {
	NameSpace                string
	GpuOperatorChannel       string
	KubeconfigPath           string
	ArtifactDir              string
	CiMachineSetInstanceType string
	CiMachineSetReplicas     string
}

var Config = config{
	NameSpace:                GetVarDefault("WORKING_NAMESPACE", "gpu-operator-test"),
	GpuOperatorChannel:       GetVarDefault("GPU_CHANNEL", ""),
	KubeconfigPath:           GetVarDefault("KUBECONFIG", ".kubeconfig"),
	ArtifactDir:              GetVarDefault("ARTIFACT_DIR", "/tmp/gpu-test"),
	CiMachineSetInstanceType: GetVarDefault("GPU_INSTANCE_TYPE", "g4dn.xlarge"),
	CiMachineSetReplicas:     GetVarDefault("GPU_REPLICAS", "1"),
}

func GetVarDefault(evar string, _default string) string {
	val := os.Getenv(evar)
	if len(val) == 0 {
		return _default
	}
	return val
}
