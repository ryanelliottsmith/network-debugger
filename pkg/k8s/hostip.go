package k8s

import (
	"context"
	"fmt"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetHostIPsForPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string, pods []types.TargetNode) ([]types.TargetNode, error) {
	var targets []types.TargetNode

	for _, pod := range pods {
		podObj, err := clientset.CoreV1().Pods(namespace).Get(ctx, pod.PodName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get pod %s: %w", pod.PodName, err)
		}

		targets = append(targets, types.TargetNode{
			NodeName:       pod.NodeName,
			PodName:        pod.PodName,
			IP:             podObj.Status.HostIP,
			IsControlPlane: pod.IsControlPlane, // Preserve control plane status from discovery
		})
	}

	return targets, nil
}
