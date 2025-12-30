package agent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

func EmitEvent(event *types.Event) error {
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}
	return nil
}

func EmitReady(self *SelfInfo, runID string) error {
	event := types.ReadyEvent(self.NodeName, "", self.PodName, runID)
	return EmitEvent(event)
}

func EmitTestStart(self *SelfInfo, check, target, runID string) error {
	event := types.TestStartEvent(self.NodeName, "", self.PodName, check, target, runID)
	return EmitEvent(event)
}

func EmitTestResult(self *SelfInfo, result *types.TestResult, runID string) error {
	status := "pass"
	if result.Status == types.StatusFail {
		status = "fail"
	}

	event := types.TestResultEvent(
		self.NodeName,
		"",
		self.PodName,
		result.Check,
		result.Target,
		status,
		result.Details,
		runID,
	)

	if result.Error != "" {
		event.Error = result.Error
	}

	return EmitEvent(event)
}

func EmitComplete(self *SelfInfo, runID string, summary interface{}) error {
	event := types.CompleteEvent(self.NodeName, "", self.PodName, summary, runID)
	return EmitEvent(event)
}

func EmitError(self *SelfInfo, runID, errMsg string) error {
	event := types.ErrorEvent(self.NodeName, "", self.PodName, errMsg, runID)
	return EmitEvent(event)
}
