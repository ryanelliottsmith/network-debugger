package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func DiscoverDaemonSetPods(ctx context.Context, clientset *kubernetes.Clientset, namespace, daemonSetName string) ([]types.TargetNode, error) {
	labelSelector := fmt.Sprintf("app=netdebug,network-mode=%s", getNetworkModeFromDaemonSetName(daemonSetName))

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	var targets []types.TargetNode
	for _, pod := range pods.Items {
		if pod.Status.Phase != "Running" {
			continue
		}

		ready := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				ready = true
				break
			}
		}

		if !ready {
			continue
		}

		targets = append(targets, types.TargetNode{
			NodeName: pod.Spec.NodeName,
			PodName:  pod.Name,
			IP:       pod.Status.PodIP,
		})
	}

	return targets, nil
}

func WaitForDaemonSetReady(ctx context.Context, clientset *kubernetes.Clientset, namespace, daemonSetName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, daemonSetName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get DaemonSet: %w", err)
		}

		if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled && ds.Status.NumberReady > 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for DaemonSet %s to be ready", daemonSetName)
}

func getNetworkModeFromDaemonSetName(daemonSetName string) string {
	if daemonSetName == "netdebug-host" {
		return "host"
	}
	return "overlay"
}
