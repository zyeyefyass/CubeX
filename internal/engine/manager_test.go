package engine

import (
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestStreamLogPipeClassification(t *testing.T) {
	mgr := &Manager{}

	type logEntry struct {
		Level   string
		Module  string
		Message string
	}
	var captured []logEntry
	mgr.SetLogCallback(func(level, module, msg string) {
		captured = append(captured, logEntry{
			Level:   level,
			Module:  module,
			Message: msg,
		})
	})

	tests := []struct {
		engine      schema.EngineType
		input       string
		wantLevel   string
		wantModule  string
		wantMsgSub  string
		shouldDrop  bool
	}{
		{
			engine:     schema.EngineSingbox,
			input:      "ERROR [2876942641 3.87s] connection: connection download closed: close tcp 172.19.0.1:59936->198.18.0.73:443: endpoint not connected\n",
			shouldDrop: true,
		},
		{
			engine:     schema.EngineSingbox,
			input:      "router: missing fakeip record, try enable experimental.cache_file\n",
			shouldDrop: true,
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
		if tc.shouldDrop {
			if len(captured) != 0 {
				t.Fatalf("expected log to be dropped for input %q, but got %d logs", tc.input, len(captured))
			}
			continue
		}
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
