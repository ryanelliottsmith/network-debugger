package agent

import (
	"fmt"
	"os"
)

type SelfInfo struct {
	NodeName string
	PodName  string
	PodIP    string
	HostIP   string
}

func GetSelfInfo() (*SelfInfo, error) {
	self := &SelfInfo{
		NodeName: os.Getenv("NODE_NAME"),
		PodName:  os.Getenv("POD_NAME"),
		PodIP:    os.Getenv("POD_IP"),
		HostIP:   os.Getenv("HOST_IP"),
	}

	if self.NodeName == "" {
		return nil, fmt.Errorf("NODE_NAME environment variable not set")
	}

	if self.PodIP == "" {
		return nil, fmt.Errorf("POD_IP environment variable not set")
	}

	if self.HostIP == "" {
		return nil, fmt.Errorf("HOST_IP environment variable not set")
	}

	return self, nil
}
