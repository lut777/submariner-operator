package network

import (
	"context"
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var kubeOvnSubnetGVR = schema.GroupVersionResource{
	Group:    "kubeovn.io",
	Version:  "v1",
	Resource: "Subnet",
}

func discoverKubeOVNNetwork(dynClient dynamic.Interface, clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	if dynClient == nil {
		return nil, nil
	}

	crClient := dynClient.Resource(kubeOvnSubnetGVR)
	crs, err := crClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {

		if apierrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.WithMessage(err, "error obtaining the KubeOVN SUBNET resources")
	}

	serviceCIDRs, err := findClusterIPRange(clientSet)
	if err != nil {
		return nil, err
	}

	return parseKubeOvnClusterNetwork(crs, serviceCIDRs), nil
}

func parseKubeOvnClusterNetwork(crs *unstructured.UnstructuredList, svcCIDR string) *ClusterNetwork {
	result := &ClusterNetwork{}
	result.PodCIDRs = []string{}
	for i, _ := range crs.Items {
		podCIDR, err := parseKubeOvnPodCIDR(&crs.Items[i])
		if err != nil {
			continue
		}

		result.PodCIDRs = append(result.PodCIDRs, podCIDR)
	}

	result.ServiceCIDRs = []string{svcCIDR}
	result.NetworkPlugin = constants.NetworkPluginCalico

	return result
}

func parseKubeOvnPodCIDR(cr *unstructured.Unstructured) (string, error) {
	podcidr, found, err := unstructured.NestedString(cr.Object, "spec", "cidrBlock")
	if err != nil {

		return "", errors.Wrap(err, "error retrieving cidr field")
	} else if !found {

		return "", errors.New("field cidr expected, but not found in subnet")
	}

	return podcidr, nil
}