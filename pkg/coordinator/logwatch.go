package coordinator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type LogWatcher struct {
	clientset *kubernetes.Clientset
	namespace string
	eventChan chan *types.Event
	errorChan chan error
	wg        sync.WaitGroup
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
	lw.wg.Add(1)
	go lw.watchPodLogs(ctx, podName)
}

func (lw *LogWatcher) watchPodLogs(ctx context.Context, podName string) {
	defer lw.wg.Done()

	podLogOpts := &corev1.PodLogOptions{
		Follow:     true,
		Timestamps: false,
	}

	req := lw.clientset.CoreV1().Pods(lw.namespace).GetLogs(podName, podLogOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		select {
		case lw.errorChan <- fmt.Errorf("failed to stream logs from pod %s: %w", podName, err):
		case <-ctx.Done():
		}
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
		select {
		case lw.errorChan <- fmt.Errorf("error reading logs from pod %s: %w", podName, err):
		case <-ctx.Done():
		}
	}
}

func (lw *LogWatcher) Close() {
	lw.wg.Wait()
	close(lw.eventChan)
	close(lw.errorChan)
}
