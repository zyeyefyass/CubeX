package main

import (
	"net"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/netstack"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestHandleIPv6PrefixChanged_SkipsWhenIPv6Disabled(t *testing.T) {
	ir := schema.NewDefaultIR()
	ir.Network.IPv6 = false

	engMgr := engine.NewManager(t.TempDir(), ir)
	srv := &Server{
		ir:     ir,
		engMgr: engMgr,
	}

	evt := netstack.IPv6AddressEvent{
		IP:        net.ParseIP("2409:8a62:2b8c:1b60:be24:11ff:fe45:62e0"),
		PrefixLen: 64,
		IsDeleted: false,
		Interface: "eth0",
	}

	if triggered := srv.handleIPv6PrefixChanged(evt); triggered {
		t.Fatalf("expected handleIPv6PrefixChanged to return false when Network.IPv6 is false")
	}

	// nil server / nil IR safety
	var nilSrv *Server
	if nilSrv.handleIPv6PrefixChanged(evt) {
		t.Fatalf("expected nil Server to return false")
	}
}
