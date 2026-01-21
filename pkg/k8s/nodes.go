package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Control plane node label keys
const (
	// Standard Kubernetes control plane label (k8s 1.24+)
	LabelControlPlane = "node-role.kubernetes.io/control-plane"
	// Legacy master label (deprecated but still common)
	LabelMaster = "node-role.kubernetes.io/master"
)

// IsControlPlaneNode checks if a node is a control plane node by examining its labels
func IsControlPlaneNode(ctx context.Context, clientset *kubernetes.Clientset, nodeName string) (bool, error) {
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	return isControlPlaneFromLabels(node.Labels), nil
}

// isControlPlaneFromLabels checks if the given labels indicate a control plane node
func isControlPlaneFromLabels(labels map[string]string) bool {
	if labels == nil {
		return false
	}

	// Check for control-plane label (value doesn't matter, just presence)
	if _, ok := labels[LabelControlPlane]; ok {
		return true
	}

	// Check for legacy master label
	if _, ok := labels[LabelMaster]; ok {
		return true
	}

	return false
}

// GetNodeRoles returns a map of node names to their control plane status
func GetNodeRoles(ctx context.Context, clientset *kubernetes.Clientset) (map[string]bool, error) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	roles := make(map[string]bool)
	for _, node := range nodes.Items {
		roles[node.Name] = isControlPlaneFromLabels(node.Labels)
	}

	return roles, nil
}
