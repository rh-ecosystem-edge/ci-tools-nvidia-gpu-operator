package ocputils

import (
	"context"
	"fmt"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1clientset "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1"
	operatorsv1alpha1clientset "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"
	pkgmanifestv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	pkgmanifestv1clientset "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned/typed/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func GetCatalogSource(config *rest.Config, namespace string, name string) (*operatorsv1alpha1.CatalogSource, error) {
	opClient, err := operatorsv1alpha1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return opClient.CatalogSources(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func GetPackageManifest(config *rest.Config, namespace string, name string) (*pkgmanifestv1.PackageManifest, error) {
	pClient, err := pkgmanifestv1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return pClient.PackageManifests(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func GetOperatorGroup(config *rest.Config, namespace string, name string) (*operatorsv1.OperatorGroup, error) {
	opClient, err := operatorsv1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return opClient.OperatorGroups(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func CreateOperatorGroup(config *rest.Config, namespace string, name string) (*operatorsv1.OperatorGroup, error) {
	opClient, err := operatorsv1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	opG := &operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:         name,
			Namespace:    namespace,
			GenerateName: fmt.Sprintf("%v-", name),
		},
		Spec: operatorsv1.OperatorGroupSpec{
			TargetNamespaces: []string{
				namespace,
			},
		},
	}
	return opClient.OperatorGroups(namespace).Create(context.TODO(), opG, metav1.CreateOptions{})
}

func CreateSubscription(config *rest.Config, namespace string, subname string,
	channel string, packageName string, catalogsource string, catalogsourceNamespace string) (*operatorsv1alpha1.Subscription, error) {
	opClient, err := operatorsv1alpha1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	sub := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name: subname,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			Channel:                channel,
			InstallPlanApproval:    "Automatic",
			CatalogSource:          catalogsource,
			CatalogSourceNamespace: catalogsourceNamespace,
			Package:                packageName,
		},
	}
	return opClient.Subscriptions(namespace).Create(context.TODO(), sub, metav1.CreateOptions{})
}

func GetSubscription(config *rest.Config, namespace string, name string) (*operatorsv1alpha1.Subscription, error) {
	opClient, err := operatorsv1alpha1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return opClient.Subscriptions(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func GetCsvByName(config *rest.Config, namespace string, name string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	opClient, err := operatorsv1alpha1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return opClient.ClusterServiceVersions(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func GetCsvsByLabel(config *rest.Config, namespace string, labelSelector string) (*operatorsv1alpha1.ClusterServiceVersionList, error) {
	opClient, err := operatorsv1alpha1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return opClient.ClusterServiceVersions(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
}

func GetCsvsByLabelAllNamespaces(config *rest.Config, labelSelector string) (*operatorsv1alpha1.ClusterServiceVersionList, error) {
	opClient, err := operatorsv1alpha1clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return opClient.ClusterServiceVersions("").List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
}

func GetAlmExamples(csv *operatorsv1alpha1.ClusterServiceVersion) (string, error) {
	almExamples := "alm-examples"
	annotations := csv.ObjectMeta.GetAnnotations()
	if example, ok := annotations[almExamples]; ok {
		return example, nil
	}
	return "", fmt.Errorf("%s not found in given csv %v", almExamples, csv)
}
