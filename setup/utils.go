package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	"ci-tools-nvidia-gpu-operator/internal"
	"ci-tools-nvidia-gpu-operator/ocputils"
	"ci-tools-nvidia-gpu-operator/testutils"
)

func waitForCsvPhase(config *rest.Config, namespace string, labelSelector string, phase operatorsv1alpha1.ClusterServiceVersionPhase) (operatorsv1alpha1.ClusterServiceVersion, error) {
	var csv operatorsv1alpha1.ClusterServiceVersion
	debugTag := fmt.Sprintf("Wait for CSV with label '%v' to become '%v'", labelSelector, phase)
	err := testutils.ExecWithRetryBackoff(debugTag, func() bool {
		csvs, err := ocputils.GetCsvsByLabel(config, internal.Config.NameSpace, labelSelector)
		if err != nil {
			return false
		}
		if len(csvs.Items) != 1 {
			return false
		}
		csv = csvs.Items[0]
		return csv.Status.Phase == phase
	}, 40, 30*time.Second)
	return csv, err
}

func getUnstructuredFromAlmExample(almExample string) (*unstructured.Unstructured, error) {
	unstructuredList := &unstructured.UnstructuredList{}
	err := json.Unmarshal([]byte(almExample), &unstructuredList.Items)
	if err != nil {
		return nil, err
	}
	if len(unstructuredList.Items) <= 0 {
		return nil, errors.New("failed to get alm examples")
	}
	return &unstructuredList.Items[0], nil
}
