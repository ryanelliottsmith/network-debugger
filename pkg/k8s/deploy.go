package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/internal/manifests"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func Install(ctx context.Context, clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, namespace, imageOverride string) error {
	// Replace namespace in all manifests
	replaceNamespace := func(yaml string) string {
		yaml = strings.ReplaceAll(yaml, "namespace: netdebug", "namespace: "+namespace)
		yaml = strings.ReplaceAll(yaml, "name: netdebug", "name: "+namespace)
		return yaml
	}

	namespaceYAML := replaceNamespace(manifests.NamespaceYAML)
	rbacYAML := replaceNamespace(manifests.RBACYAML)
	configMapYAML := replaceNamespace(manifests.ConfigMapYAML)
	hostDS := replaceNamespace(manifests.DaemonSetHostYAML)
	overlayDS := replaceNamespace(manifests.DaemonSetOverlayYAML)

	if imageOverride != "" {
		hostDS = strings.ReplaceAll(hostDS, "ghcr.io/ryanelliottsmith/network-debugger:latest", imageOverride)
		overlayDS = strings.ReplaceAll(overlayDS, "ghcr.io/ryanelliottsmith/network-debugger:latest", imageOverride)
	}

	if err := applyYAML(ctx, dynamicClient, namespaceYAML); err != nil {
		return fmt.Errorf("failed to apply namespace: %w", err)
	}

	if err := applyYAML(ctx, dynamicClient, rbacYAML); err != nil {
		return fmt.Errorf("failed to apply RBAC: %w", err)
	}

	if err := applyYAML(ctx, dynamicClient, configMapYAML); err != nil {
		return fmt.Errorf("failed to apply ConfigMap: %w", err)
	}

	if err := applyYAML(ctx, dynamicClient, hostDS); err != nil {
		return fmt.Errorf("failed to apply host DaemonSet: %w", err)
	}

	if err := applyYAML(ctx, dynamicClient, overlayDS); err != nil {
		return fmt.Errorf("failed to apply overlay DaemonSet: %w", err)
	}

	return nil
}

func Uninstall(ctx context.Context, dynamicClient dynamic.Interface, namespace string) error {
	// Replace namespace in all manifests
	replaceNamespace := func(yaml string) string {
		yaml = strings.ReplaceAll(yaml, "namespace: netdebug", "namespace: "+namespace)
		yaml = strings.ReplaceAll(yaml, "name: netdebug", "name: "+namespace)
		return yaml
	}

	manifestList := []string{
		replaceNamespace(manifests.DaemonSetOverlayYAML),
		replaceNamespace(manifests.DaemonSetHostYAML),
		replaceNamespace(manifests.ConfigMapYAML),
		replaceNamespace(manifests.RBACYAML),
		replaceNamespace(manifests.NamespaceYAML),
	}

	for _, manifest := range manifestList {
		if err := deleteYAML(ctx, dynamicClient, manifest); err != nil {
			fmt.Printf("Warning: failed to delete resource: %v\n", err)
		}
	}

	return nil
}

func applyYAML(ctx context.Context, dynamicClient dynamic.Interface, yamlContent string) error {
	docs := strings.Split(yamlContent, "---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), obj); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		gvr := schema.GroupVersionResource{
			Group:    obj.GroupVersionKind().Group,
			Version:  obj.GroupVersionKind().Version,
			Resource: getResourceName(obj.GetKind()),
		}

		namespace := obj.GetNamespace()
		var err error

		if namespace != "" {
			_, err = dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
			if err != nil && strings.Contains(err.Error(), "already exists") {
				_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
			}
		} else {
			_, err = dynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
			if err != nil && strings.Contains(err.Error(), "already exists") {
				_, err = dynamicClient.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
			}
		}

		if err != nil {
			return fmt.Errorf("failed to apply %s %s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}

	return nil
}

func deleteYAML(ctx context.Context, dynamicClient dynamic.Interface, yamlContent string) error {
	docs := strings.Split(yamlContent, "---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), obj); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		gvr := schema.GroupVersionResource{
			Group:    obj.GroupVersionKind().Group,
			Version:  obj.GroupVersionKind().Version,
			Resource: getResourceName(obj.GetKind()),
		}

		namespace := obj.GetNamespace()
		var err error

		if namespace != "" {
			err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
		} else {
			err = dynamicClient.Resource(gvr).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
		}

		if err != nil && !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to delete %s %s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}

	return nil
}

func getResourceName(kind string) string {
	kind = strings.ToLower(kind)
	switch kind {
	case "namespace":
		return "namespaces"
	case "serviceaccount":
		return "serviceaccounts"
	case "clusterrole":
		return "clusterroles"
	case "clusterrolebinding":
		return "clusterrolebindings"
	case "configmap":
		return "configmaps"
	case "daemonset":
		return "daemonsets"
	default:
		return kind + "s"
	}
}
