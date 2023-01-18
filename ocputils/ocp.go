package ocputils

import (
	"context"
	"encoding/json"

	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type serverVersion struct {
	Kubernetes *version.Info
	Openshift  *utilversion.Version
}

func GetServerVersion(config *rest.Config) (*serverVersion, error) {
	oClient, err := configv1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	clusterVersion, err := oClient.ClusterVersions().Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var ocpVersion *utilversion.Version = nil
	for _, condition := range clusterVersion.Status.History {
		if condition.State != "Completed" {
			continue
		}

		ocpVersion, err = utilversion.ParseGeneric(condition.Version)
		if err != nil {
			return nil, err
		}
		break
	}
	discovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	k8sversion, err := discovery.ServerVersion()
	if err != nil {
		return nil, err
	}
	return &serverVersion{
		Kubernetes: k8sversion,
		Openshift:  ocpVersion,
	}, nil
}

func CreateDynamicResource(config *rest.Config, resource schema.GroupVersionResource, obj runtime.Object, namespace string) (*unstructured.Unstructured, error) {
	dClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{} = map[string]interface{}{}
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	object := &unstructured.Unstructured{
		Object: m,
	}
	return dClient.Resource(resource).Namespace(namespace).Create(context.TODO(), object, metav1.CreateOptions{})
}

func ListDynamicResource(config *rest.Config, resource schema.GroupVersionResource) (*unstructured.UnstructuredList, error) {
	dClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return dClient.Resource(resource).List(context.TODO(), metav1.ListOptions{})
}

func GetDynamicResource[T runtime.Object](config *rest.Config, resource schema.GroupVersionResource, namespace string, name string, obj T) error {
	dClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	resp, err := dClient.Resource(resource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(resp.UnstructuredContent(), obj)
}
