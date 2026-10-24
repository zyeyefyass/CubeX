package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestManagerPreFlightAndBinaryDiscovery(t *testing.T) {
	workDir := filepath.Join("..", "..")
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		t.Fatalf("failed to get abs workdir: %v", err)
	}

	ir := schema.NewDefaultIR()
	mgr := NewManager(absWorkDir, ir)

	mihomoBin := mgr.findBinary("mihomo")
	if _, err := os.Stat(mihomoBin); err != nil {
		t.Errorf("failed to discover mihomo binary: %s (%v)", mihomoBin, err)
	}

	singboxBin := mgr.findBinary("sing-box")
	if _, err := os.Stat(singboxBin); err != nil {
		t.Errorf("failed to discover sing-box binary: %s (%v)", singboxBin, err)
	}

	etcDir := filepath.Join(absWorkDir, "etc")
	mihomoConfig := filepath.Join(etcDir, "config.yaml")
	mihomoData, err := compiler.BuildMihomo(ir)
	if err != nil {
		t.Fatalf("failed to compile mihomo: %v", err)
	}
	_ = os.WriteFile(mihomoConfig, mihomoData, 0644)

	if err := mgr.dryRunCheck(schema.EngineMihomo, mihomoBin, etcDir, mihomoConfig); err != nil {
		t.Errorf("mihomo dry-run validation failed: %v", err)
	}

	singboxConfig := filepath.Join(etcDir, "config.json")
	singboxData, err := compiler.BuildSingBox(ir)
	if err != nil {
		t.Fatalf("failed to compile sing-box: %v", err)
	}
	_ = os.WriteFile(singboxConfig, singboxData, 0644)

	if err := mgr.dryRunCheck(schema.EngineSingbox, singboxBin, etcDir, singboxConfig); err != nil {
		t.Errorf("sing-box dry-run validation failed: %v", err)
	}
}

func TestStreamLogPipeClassification(t *testing.T) {
	mgr := NewManager(".", schema.NewDefaultIR())

	var captured []struct {
		Level   string
		Module  string
		Message string
	}

	mgr.SetLogCallback(func(level, module, msg string) {
		captured = append(captured, struct {
			Level   string
			Module  string
			Message string
		}{Level: level, Module: module, Message: msg})
	})

	tests := []struct {
		engine      schema.EngineType
		input       string
		wantLevel   string
		wantModule  string
		wantMsgSub  string
	}{
		{
			engine:     schema.EngineSingbox,
			input:      "ERROR [2876942641 3.87s] connection: connection download closed: close tcp 172.19.0.1:59936->198.18.0.73:443: endpoint not connected\n",
			wantLevel:  "DEBUG",
			wantModule: "SINGBOX",
			wantMsgSub: "endpoint not connected",
		},
		{
			engine:     schema.EngineSingbox,
			input:      "ERROR [1276080763 267ms] connection: report handshake success: connection refused\n",
			wantLevel:  "DEBUG",
			wantModule: "SINGBOX",
			wantMsgSub: "connection refused",
		},
		{
			engine:     schema.EngineSingbox,
			input:      "ERROR [1742375921 17.86s] connection: connection upload closed: read tcp 172.19.0.1:16956->198.18.0.13:443: software caused connection abort\n",
			wantLevel:  "DEBUG",
			wantModule: "SINGBOX",
			wantMsgSub: "software caused connection abort",
		},
		{
			engine:     schema.EngineMihomo,
			input:      `time="2026-10-22T21:47:09.917846991+08:00" level=warning msg="[TUN] default interface changed by monitor, => veth-cubex"` + "\n",
			wantLevel:  "DEBUG",
			wantModule: "MIHOMO",
			wantMsgSub: "[TUN] default interface changed by monitor, => veth-cubex",
		},
		{
			engine:     schema.EngineSingbox,
			input:      "FATAL sing-box crashed: cannot parse configuration json\n",
			wantLevel:  "ERROR",
			wantModule: "SINGBOX",
			wantMsgSub: "sing-box crashed",
		},
		{
			engine:     schema.EngineMihomo,
			input:      `time="2026-10-22T21:47:09.917846991+08:00" level=warning msg="deprecated rule set format"` + "\n",
			wantLevel:  "WARN",
			wantModule: "MIHOMO",
			wantMsgSub: "deprecated rule set format",
		},
		{
			engine:     schema.EngineMihomo,
			input:      `time="2026-10-24T10:09:26Z" level=warning msg="Mihomo shutting down"` + "\n",
			wantLevel:  "INFO",
			wantModule: "MIHOMO",
			wantMsgSub: "Mihomo shutting down",
		},
	}

	for _, tc := range tests {
		captured = nil
		mgr.streamLogPipe(tc.engine, "stdout", strings.NewReader(tc.input))
		if len(captured) != 1 {
			t.Fatalf("expected 1 log captured for input %q, got %d", tc.input, len(captured))
		}
		got := captured[0]
		if got.Level != tc.wantLevel {
			t.Errorf("input %q: got Level %s, want %s", tc.input, got.Level, tc.wantLevel)
		}
		if got.Module != tc.wantModule {
			t.Errorf("input %q: got Module %s, want %s", tc.input, got.Module, tc.wantModule)
		}
		if !strings.Contains(got.Message, tc.wantMsgSub) {
			t.Errorf("input %q: got Message %q, expected to contain %q", tc.input, got.Message, tc.wantMsgSub)
		}
	}
}
