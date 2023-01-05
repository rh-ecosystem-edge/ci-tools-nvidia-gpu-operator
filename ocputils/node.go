package ocputils

import (
	"context"
	"fmt"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machinev1beta1client "github.com/openshift/client-go/machine/clientset/versioned/typed/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetNodesByLabelSelector(config *rest.Config, labelSelector string) (*corev1.NodeList, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}
func GetNodesByLabel(config *rest.Config, labelselector string) (*v1.NodeList, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelselector,
	})
}

func GetNodesByRole(config *rest.Config, role string) (*v1.NodeList, error) {
	return GetNodesByLabel(config, fmt.Sprintf("node-role.kubernetes.io/%v", role))
}

func GetFirstWorkerNode(config *rest.Config) (*v1.Node, error) {
	nodes, err := GetNodesByRole(config, "worker")
	if err != nil {
		return nil, err
	}
	return &nodes.Items[0], nil
}

func GetWorkerMachineSets(config *rest.Config, namespace string) (*machinev1beta1.MachineSetList, error) {
	clientset, err := machinev1beta1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	list := &machinev1beta1.MachineSetList{
		Items: []machinev1beta1.MachineSet{},
	}
	resp, err := clientset.MachineSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ms := range resp.Items {
		if val, ok := ms.Spec.Template.ObjectMeta.Labels["machine.openshift.io/cluster-api-machine-role"]; ok && val == "worker" {
			list.Items = append(list.Items, ms)
		}
	}
	return list, nil
}

func PatchMachineSet(config *rest.Config, ms *machinev1beta1.MachineSet, data []byte, pt types.PatchType) (*machinev1beta1.MachineSet, error) {
	clientset, err := machinev1beta1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.MachineSets(ms.Namespace).Patch(context.TODO(), ms.Name, pt, data, metav1.PatchOptions{})
}

func GetMachineSet(config *rest.Config, namespace string, name string) (*machinev1beta1.MachineSet, error) {
	clientset, err := machinev1beta1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.MachineSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func CreateMachineSet(config *rest.Config, namespace string, ms *machinev1beta1.MachineSet) (*machinev1beta1.MachineSet, error) {
	clientset, err := machinev1beta1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.MachineSets(namespace).Create(context.TODO(), ms, metav1.CreateOptions{})
}
