package host

import (
	"testing"
)

func TestTopologyManager(t *testing.T) {
	mgr := GetTopologyManager()
	graph := mgr.Refresh()

	if len(graph.Nodes) < 4 {
		t.Errorf("expected at least 4 nodes (WAN, Router, CubeX, Switch), got %d", len(graph.Nodes))
	}
	if len(graph.Links) < 3 {
		t.Errorf("expected at least 3 links, got %d", len(graph.Links))
	}

	cached := mgr.GetGraph()
	if cached.TotalNodes != len(graph.Nodes) {
		t.Errorf("expected cached nodes to match refreshed nodes")
	}
}
