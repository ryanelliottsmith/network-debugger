package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

func Run(ctx context.Context, mode, configRef string) error {
	log.Printf("Starting agent in %s mode", mode)

	self, err := GetSelfInfo()
	if err != nil {
		return fmt.Errorf("failed to get self info: %w", err)
	}

	log.Printf("Agent info: node=%s, pod=%s, podIP=%s, hostIP=%s",
		self.NodeName, self.PodName, self.PodIP, self.HostIP)

	if err := StartIperf3Server(); err != nil {
		log.Printf("WARNING: Failed to start iperf3 server: %v", err)
		log.Printf("Bandwidth tests will be skipped on this node")
	} else {
		log.Printf("iperf3 server started successfully")
	}

	if mode == "configmap" {
		return runConfigMapMode(ctx, self, configRef)
	}

	return fmt.Errorf("direct mode not yet implemented")
}

func runConfigMapMode(ctx context.Context, self *SelfInfo, configRef string) error {
	log.Printf("Starting ConfigMap watch mode")

	namespace, configMapName, err := parseConfigRef(configRef)
	if err != nil {
		return fmt.Errorf("invalid config reference: %w", err)
	}

	log.Printf("Watching ConfigMap: %s/%s", namespace, configMapName)

	return WatchConfigMap(ctx, namespace, configMapName, func(config *types.Config) error {
		log.Printf("Handling new run: %s", config.RunID)

		if err := RunTests(ctx, config, self); err != nil {
			log.Printf("Error running tests: %v", err)

			if emitErr := EmitError(self, config.RunID, err.Error()); emitErr != nil {
				log.Printf("Failed to emit error event: %v", emitErr)
			}

			return err
		}

		return nil
	})
}

func parseConfigRef(configRef string) (namespace, name string, err error) {
	parts := strings.Split(configRef, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("config must be in format NAMESPACE/CONFIGMAPNAME, got: %s", configRef)
	}
	return parts[0], parts[1], nil
}
