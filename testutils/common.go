package testutils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"

	"ci-tools-nvidia-gpu-operator/internal"
)

type TestFunc func() bool

func Printf(debugTag string, format string, a ...any) {
	str := fmt.Sprintf(format, a...)
	fmt.Printf("[%v]: %v\n", debugTag, str)
	ginkgo.AddReportEntry(debugTag, str)
}

func ExecWithRetryBackoff(debugTag string, fn TestFunc, maxRetries int, interval time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		if fn() {
			return nil
		}
		Printf("Retry loop: ", "[%v] attempt %d/%d failed.", debugTag, i+1, maxRetries)
		time.Sleep(interval)
	}
	return fmt.Errorf("Max retries exceeded. Max retries was set to %v", maxRetries)
}

func SaveToArtifactsDir(data []byte, filename string) error {
	filepath := path.Join(internal.Config.ArtifactDir, filename)
	Printf("SaveToArtifactsDir", "Writing data to file: %v", filepath)
	return ioutil.WriteFile(filepath, data, 0644)
}

func SaveAsJsonToArtifactsDir(obj interface{}, filename string) error {
	jsn, err := json.MarshalIndent(obj, "", " ")
	if err != nil {
		return err
	}
	return SaveToArtifactsDir(jsn, filename)
}

func SkipTestIfEnvVarSet(envVar string, isSet bool) bool {
	envVarVal := os.Getenv(envVar)
	skippedMsg := fmt.Sprintf("Skipped due to %v=%v", envVar, envVarVal)
	if isSet && len(envVarVal) != 0 {
		ginkgo.Skip(skippedMsg)
		return true
	}
	if !isSet && len(envVarVal) == 0 {
		ginkgo.Skip(skippedMsg)
		return true
	}
	return false
}
