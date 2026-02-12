package checks

import (
	"context"
	"net"
	"testing"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

func TestPortsCheck_UDP(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start UDP listener: %v", err)
	}
	defer pc.Close()

	addr := pc.LocalAddr().(*net.UDPAddr)
	openPort := addr.Port

	go func() {
		buf := make([]byte, 1024)
		for {
			n, remoteAddr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			// Echo back
			pc.WriteTo(buf[:n], remoteAddr)
		}
	}()

	closedPort := 50000 + (openPort % 1000)

	check := NewPortsCheck([]types.PortCheck{
		{Port: openPort, Protocol: "udp", Name: "open-udp-service", NodeRole: types.NodeRoleAll},
		{Port: closedPort, Protocol: "udp", Name: "closed-udp-service", NodeRole: types.NodeRoleAll},
	})

	ctx := context.Background()
	result, err := check.Run(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Check run failed: %v", err)
	}

	portResults := result.Details["ports"].([]types.PortCheckDetails)
	var openResult, closedResult types.PortCheckDetails

	for _, res := range portResults {
		if res.Port == openPort {
			openResult = res
		} else if res.Port == closedPort {
			closedResult = res
		}
	}

	if !openResult.Open {
		t.Errorf("Expected UDP port %d to be OPEN, but got CLOSED", openPort)
	}

	// Verify Closed Port
	if closedResult.Open {
		t.Errorf("Expected UDP port %d to be CLOSED, but got OPEN", closedPort)
	} else {
		t.Logf("UDP port %d correctly reported as CLOSED", closedPort)
		if closedResult.Error == "" {
			t.Error("Expected error message for closed port, but got empty string")
		}
	}

	// 6. Verify Silent/Drop Port (should be CLOSED due to timeout)
	// We'll use a listener that reads but never writes back
	silentConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start silent UDP listener: %v", err)
	}
	defer silentConn.Close()
	silentPort := silentConn.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf := make([]byte, 1024)
		for {
			_, _, err := silentConn.ReadFrom(buf)
			if err != nil {
				return
			}
			// Do NOT write back
		}
	}()

	silentCheck := NewPortsCheck([]types.PortCheck{
		{Port: silentPort, Protocol: "udp", Name: "silent-udp-service", NodeRole: types.NodeRoleAll},
	})

	silentResult, err := silentCheck.Run(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Silent check run failed: %v", err)
	}

	silentPortResult := silentResult.Details["ports"].([]types.PortCheckDetails)[0]
	if silentPortResult.Open {
		t.Errorf("Expected silent UDP port %d to be CLOSED (timeout), but got OPEN", silentPort)
	} else {
		t.Logf("Silent UDP port %d correctly reported as CLOSED", silentPort)
		if silentPortResult.Error == "" {
			t.Error("Expected timeout error message for silent port, but got empty string")
		} else {
			t.Logf("Silent port error: %s", silentPortResult.Error)
		}
	}
}

func TestPortsCheck_TCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen tcp: %v", err)
	}
	defer ln.Close()
	openPort := ln.Addr().(*net.TCPAddr).Port

	closedPort := 51000 + (openPort % 1000)

	check := NewPortsCheck([]types.PortCheck{
		{Port: openPort, Protocol: "tcp", Name: "open-tcp", NodeRole: types.NodeRoleAll},
		{Port: closedPort, Protocol: "tcp", Name: "closed-tcp", NodeRole: types.NodeRoleAll},
	})

	result, err := check.Run(context.Background(), "127.0.0.1")
	if err != nil {
		t.Fatalf("Check run failed: %v", err)
	}

	portResults := result.Details["ports"].([]types.PortCheckDetails)
	for _, res := range portResults {
		if res.Port == openPort && !res.Open {
			t.Errorf("Expected TCP port %d to be OPEN", openPort)
		}
		if res.Port == closedPort && res.Open {
			t.Errorf("Expected TCP port %d to be CLOSED", closedPort)
		}
	}
}
