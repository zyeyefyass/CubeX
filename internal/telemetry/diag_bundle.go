package telemetry

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"time"
)

// DiagBundlePayload 定义一键导出的脱敏诊断包结构
type DiagBundlePayload struct {
	ExportedAt    time.Time     `json:"exported_at"`
	GoVersion     string        `json:"go_version"`
	Arch          string        `json:"arch"`
	HeapAllocKB   uint64        `json:"heap_alloc_kb"`
	SysMemKB      uint64        `json:"sys_mem_kb"`
	NumGoroutine  int           `json:"num_goroutine"`
	ActiveEngine  string        `json:"active_engine"`
	FlightEvents  []FlightEvent `json:"flight_events"`
	RedactedIRRaw string        `json:"redacted_ir_summary"`
}

var sensitiveRegex = regexp.MustCompile(`(?i)("?(?:password|uuid|secret|token|private_key|auth|sub_url)"?\s*:\s*")([^"]+)(")`)

// RedactSensitiveJSON 将 JSON/YAML 文本中的密码、UUID、Token、订阅链接自动脱敏为 ***REDACTED***
func RedactSensitiveJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return string(sensitiveRegex.ReplaceAll(raw, []byte(`${1}***REDACTED***${3}`)))
}

// ServeDiagnosticBundle 挂载于 GET /api/v1/diag/bundle，流式输出 .json.gz 诊断包（0 磁盘写入）
func ServeDiagnosticBundle(
	recorder *FlightRecorder,
	activeEngineFn func() string,
	rawIRSnapshotFn func() []byte,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)

		var events []FlightEvent
		if recorder != nil {
			events = recorder.Snapshot(RingBufferCapacity)
		}

		var redactedIR string
		if rawIRSnapshotFn != nil {
			redactedIR = RedactSensitiveJSON(rawIRSnapshotFn())
		}

		engineName := ""
		if activeEngineFn != nil {
			engineName = activeEngineFn()
		}

		bundle := DiagBundlePayload{
			ExportedAt:    time.Now(),
			GoVersion:     runtime.Version(),
			Arch:          runtime.GOARCH,
			HeapAllocKB:   ms.HeapAlloc / 1024,
			SysMemKB:      ms.Sys / 1024,
			NumGoroutine:  runtime.NumGoroutine(),
			ActiveEngine:  engineName,
			FlightEvents:  events,
			RedactedIRRaw: redactedIR,
		}

		filename := fmt.Sprintf("cubex-diag-%s.json.gz", time.Now().Format("20060102-150405"))
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.Header().Set("Cache-Control", "no-store")

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, "gzip init failed", http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		enc := json.NewEncoder(gz)
		enc.SetIndent("", "  ")
		_ = enc.Encode(bundle)
	}
}
