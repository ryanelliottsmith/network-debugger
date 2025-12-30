package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Coordinator struct {
	clientset *kubernetes.Clientset
	namespace string
	configMap string
}

func NewCoordinator(clientset *kubernetes.Clientset, namespace, configMap string) *Coordinator {
	return &Coordinator{
		clientset: clientset,
		namespace: namespace,
		configMap: configMap,
	}
}

func (c *Coordinator) UpdateConfig(ctx context.Context, config *types.Config) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	cm, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.configMap, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["config.json"] = string(configJSON)

	_, err = c.clientset.CoreV1().ConfigMaps(c.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	return nil
}

func (c *Coordinator) RunTests(ctx context.Context, config *types.Config, podNames []string, timeout time.Duration) ([]*types.Event, error) {
	if err := c.UpdateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	watcher := NewLogWatcher(c.clientset, c.namespace)
	defer watcher.Close()

	agg := NewAggregator(podNames)

	for _, podName := range podNames {
		watcher.WatchPod(ctx, podName)
	}

	testCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	readyTimeout := time.After(30 * time.Second)
	readyTicker := time.NewTicker(500 * time.Millisecond)
	defer readyTicker.Stop()

readyLoop:
	for {
		select {
		case event := <-watcher.EventChan():
			if event.RunID == config.RunID {
				agg.AddEvent(event)
				if agg.AllPodsReady() {
					break readyLoop
				}
			}
		case err := <-watcher.ErrorChan():
			fmt.Printf("Warning: log watcher error: %v\n", err)
		case <-readyTimeout:
			return nil, fmt.Errorf("timeout waiting for pods to be ready (%d/%d ready)", agg.GetReadyCount(), agg.GetExpectedCount())
		case <-testCtx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for pods to be ready")
		case <-readyTicker.C:
			if agg.AllPodsReady() {
				break readyLoop
			}
		}
	}

	completeTicker := time.NewTicker(500 * time.Millisecond)
	defer completeTicker.Stop()

	for {
		select {
		case event := <-watcher.EventChan():
			if event.RunID == config.RunID {
				agg.AddEvent(event)
				if agg.AllPodsComplete() {
					return agg.GetEvents(), nil
				}
			}
		case err := <-watcher.ErrorChan():
			fmt.Printf("Warning: log watcher error: %v\n", err)
		case <-testCtx.Done():
			return agg.GetEvents(), fmt.Errorf("timeout waiting for tests to complete (%d/%d complete)", agg.GetCompletedCount(), agg.GetExpectedCount())
		case <-completeTicker.C:
			if agg.AllPodsComplete() {
				return agg.GetEvents(), nil
			}
		}
	}
}

func GenerateRunID() string {
	return uuid.New().String()
}

func GenerateBandwidthPairs(targets []types.TargetNode) [][2]types.TargetNode {
	var pairs [][2]types.TargetNode

	for i := 0; i < len(targets); i++ {
		for j := i + 1; j < len(targets); j++ {
			pairs = append(pairs, [2]types.TargetNode{targets[i], targets[j]})
		}
	}

	return pairs
}
