package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/k8s"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manage DaemonSet deployment",
	Long:  "Install, uninstall, or check status of the network debugger DaemonSet.",
}

var deployInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Deploy DaemonSet and RBAC resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		imageOverride, _ := cmd.Flags().GetString("image")

		ctx := context.Background()

		fmt.Printf("Installing netdebug in namespace: %s\n", namespace)

		clientset, err := k8s.GetClientset()
		if err != nil {
			return fmt.Errorf("failed to create kubernetes client: %w", err)
		}

		dynamicClient, err := k8s.GetDynamicClient()
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		if err := k8s.Install(ctx, clientset, dynamicClient, namespace, imageOverride); err != nil {
			return fmt.Errorf("failed to install: %w", err)
		}

		fmt.Println("✓ Resources deployed successfully")

		fmt.Println("\nWaiting for DaemonSets to be ready...")

		if err := k8s.WaitForDaemonSetReady(ctx, clientset, namespace, "netdebug-host", 2*time.Minute); err != nil {
			fmt.Printf("Warning: Host DaemonSet not ready: %v\n", err)
		} else {
			fmt.Println("✓ Host network DaemonSet ready")
		}

		if err := k8s.WaitForDaemonSetReady(ctx, clientset, namespace, "netdebug-overlay", 2*time.Minute); err != nil {
			fmt.Printf("Warning: Overlay DaemonSet not ready: %v\n", err)
		} else {
			fmt.Println("✓ Overlay network DaemonSet ready")
		}

		fmt.Printf("\nInstallation complete! Use 'netdebug run' to start testing.\n")

		return nil
	},
}

var deployUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove DaemonSet and RBAC resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")

		ctx := context.Background()

		fmt.Printf("Uninstalling netdebug from namespace: %s\n", namespace)

		dynamicClient, err := k8s.GetDynamicClient()
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		if err := k8s.Uninstall(ctx, dynamicClient, namespace); err != nil {
			return fmt.Errorf("failed to uninstall: %w", err)
		}

		fmt.Println("✓ Resources removed successfully")

		return nil
	},
}

var deployTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Output Kubernetes manifests to stdout",
	Long:  "Output all Kubernetes manifests to stdout for customization before applying.\n\nExample:\n  netdebug deploy template > manifests.yaml\n  netdebug deploy template --namespace my-ns --image myrepo/netdebug:v1.0 > manifests.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		imageOverride, _ := cmd.Flags().GetString("image")

		fmt.Print(k8s.GetAllManifests(namespace, imageOverride))
		return nil
	},
}

var deployStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check DaemonSet deployment status",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")

		ctx := context.Background()

		clientset, err := k8s.GetClientset()
		if err != nil {
			return fmt.Errorf("failed to create kubernetes client: %w", err)
		}

		fmt.Printf("Checking status in namespace: %s\n\n", namespace)

		hostDS, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, "netdebug-host", metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Host DaemonSet: Not found\n")
		} else {
			fmt.Printf("Host DaemonSet:\n")
			fmt.Printf("  Desired: %d\n", hostDS.Status.DesiredNumberScheduled)
			fmt.Printf("  Ready:   %d\n", hostDS.Status.NumberReady)
			fmt.Printf("  Available: %d\n", hostDS.Status.NumberAvailable)
		}

		overlayDS, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, "netdebug-overlay", metav1.GetOptions{})
		if err != nil {
			fmt.Printf("\nOverlay DaemonSet: Not found\n")
		} else {
			fmt.Printf("\nOverlay DaemonSet:\n")
			fmt.Printf("  Desired: %d\n", overlayDS.Status.DesiredNumberScheduled)
			fmt.Printf("  Ready:   %d\n", overlayDS.Status.NumberReady)
			fmt.Printf("  Available: %d\n", overlayDS.Status.NumberAvailable)
		}

		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=netdebug",
		})
		if err == nil && len(pods.Items) > 0 {
			fmt.Printf("\nPods:\n")
			for _, pod := range pods.Items {
				status := string(pod.Status.Phase)
				ready := "Not Ready"
				for _, condition := range pod.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "True" {
						ready = "Ready"
						break
					}
				}
				fmt.Printf("  %s (%s) - %s - %s\n", pod.Name, pod.Spec.NodeName, status, ready)
			}
		}

		return nil
	},
}

func init() {
	deployCmd.AddCommand(deployInstallCmd)
	deployCmd.AddCommand(deployUninstallCmd)
	deployCmd.AddCommand(deployStatusCmd)
	deployCmd.AddCommand(deployTemplateCmd)

	for _, cmd := range []*cobra.Command{deployInstallCmd, deployUninstallCmd, deployStatusCmd, deployTemplateCmd} {
		cmd.Flags().StringP("namespace", "n", "default", "Namespace for deployment")
	}

	deployInstallCmd.Flags().String("image", "", "Override default image (default: ghcr.io/ryanelliottsmith/network-debugger:"+version+")")
	deployTemplateCmd.Flags().String("image", "", "Override default image (default: ghcr.io/ryanelliottsmith/network-debugger:"+version+")")
}
