package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func WatchConfigMap(ctx context.Context, namespace, configMapName string, handler func(*types.Config) error) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	var lastRunID string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		watcher, err := clientset.CoreV1().ConfigMaps(namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", configMapName),
		})
		if err != nil {
			log.Printf("Failed to watch ConfigMap: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for event := range watcher.ResultChan() {
			if event.Type == watch.Added || event.Type == watch.Modified {
				cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
				if err != nil {
					log.Printf("Failed to get ConfigMap: %v", err)
					continue
				}

				configJSON, ok := cm.Data["config.json"]
				if !ok {
					log.Printf("ConfigMap does not contain config.json")
					continue
				}

				var config types.Config
				if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
					log.Printf("Failed to parse config.json: %v", err)
					continue
				}

				if config.RunID != "" && config.RunID != lastRunID {
					log.Printf("New run detected: %s", config.RunID)
					lastRunID = config.RunID

					if err := handler(&config); err != nil {
						log.Printf("Handler error: %v", err)
					}
				}
			}
		}

		log.Printf("Watch channel closed, reconnecting in 2s...")
		time.Sleep(2 * time.Second)
	}
}
