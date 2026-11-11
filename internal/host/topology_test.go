package host

import (
	"testing"
)

func TestTopologyManager(t *testing.T) {
	mgr := GetTopologyManager()
	graph := mgr.Refresh()

	if len(graph.Nodes) < 3 {
		t.Errorf("expected at least 3 core nodes (WAN, Router, CubeX), got %d", len(graph.Nodes))
	}
	if len(graph.Links) < 2 {
		t.Errorf("expected at least 2 core links, got %d", len(graph.Links))
	}

	cached := mgr.GetGraph()
	if cached.TotalNodes != len(graph.Nodes) {
		t.Errorf("expected cached nodes to match refreshed nodes")
	}
}
