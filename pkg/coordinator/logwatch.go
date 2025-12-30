package coordinator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type LogWatcher struct {
	clientset *kubernetes.Clientset
	namespace string
	eventChan chan *types.Event
	errorChan chan error
}

func NewLogWatcher(clientset *kubernetes.Clientset, namespace string) *LogWatcher {
	return &LogWatcher{
		clientset: clientset,
		namespace: namespace,
		eventChan: make(chan *types.Event, 100),
		errorChan: make(chan error, 10),
	}
}

func (lw *LogWatcher) EventChan() <-chan *types.Event {
	return lw.eventChan
}

func (lw *LogWatcher) ErrorChan() <-chan error {
	return lw.errorChan
}

func (lw *LogWatcher) WatchPod(ctx context.Context, podName string) {
	go lw.watchPodLogs(ctx, podName)
}

func (lw *LogWatcher) watchPodLogs(ctx context.Context, podName string) {
	podLogOpts := &corev1.PodLogOptions{
		Follow:     true,
		Timestamps: false,
	}

	req := lw.clientset.CoreV1().Pods(lw.namespace).GetLogs(podName, podLogOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		lw.errorChan <- fmt.Errorf("failed to stream logs from pod %s: %w", podName, err)
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()

		var event types.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		select {
		case lw.eventChan <- &event:
		case <-ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		lw.errorChan <- fmt.Errorf("error reading logs from pod %s: %w", podName, err)
	}
}

func (lw *LogWatcher) Close() {
	close(lw.eventChan)
	close(lw.errorChan)
}
