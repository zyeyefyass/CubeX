package telemetry

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactSensitiveJSON(t *testing.T) {
	raw := []byte(`{
		"name": "Node-1",
		"password": "super-secret-password-123",
		"uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"token": "tok_abcdef123456",
		"private_key": "MIIEvgIBADANBgkqhkiG9w0BAQEFAASC",
		"sub_url": "https://example.com/api/v1/client/subscribe?token=secret123"
	}`)

	redacted := RedactSensitiveJSON(raw)

	// 1. 验证敏感字段被替换为 ***REDACTED***
	if strings.Contains(redacted, "super-secret-password-123") {
		t.Errorf("password leaked in redacted text")
	}
	if strings.Contains(redacted, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("uuid leaked in redacted text")
	}
	if strings.Contains(redacted, "tok_abcdef123456") {
		t.Errorf("token leaked in redacted text")
	}
	if strings.Contains(redacted, "MIIEvgIBADANBgkqhkiG9w0BAQEFAASC") {
		t.Errorf("private_key leaked in redacted text")
	}
	if strings.Contains(redacted, "secret123") {
		t.Errorf("sub_url leaked in redacted text")
	}

	// 2. 验证非敏感字段保持原样
	if !strings.Contains(redacted, "Node-1") {
		t.Errorf("non-sensitive field was corrupted")
	}
}

func TestServeDiagnosticBundleGzipStream(t *testing.T) {
	rec := NewFlightRecorder()
	rec.Record(CatEngineSwitch, "INFO", "Switch to sing-box", 45)

	handler := ServeDiagnosticBundle(
		rec,
		func() string { return "singbox" },
		func() []byte { return []byte(`{"uuid":"secret-node-id","server":"1.2.3.4"}`) },
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/diag/bundle", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "application/gzip" {
		t.Errorf("expected Content-Type application/gzip, got %s", resp.Header.Get("Content-Type"))
	}

	// 解压 Gzip 流并验证 JSON 结构
	gzReader, err := gzip.NewReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	var payload DiagBundlePayload
	if err := json.NewDecoder(gzReader).Decode(&payload); err != nil {
		t.Fatalf("failed to decode gzip payload JSON: %v", err)
	}

	if payload.ActiveEngine != "singbox" {
		t.Errorf("expected active engine singbox, got %s", payload.ActiveEngine)
	}
	if len(payload.FlightEvents) != 1 {
		t.Fatalf("expected 1 flight event, got %d", len(payload.FlightEvents))
	}
	if strings.Contains(payload.RedactedIRRaw, "secret-node-id") {
		t.Errorf("leaked secret-node-id in diagnostic payload")
	}
	if !strings.Contains(payload.RedactedIRRaw, "***REDACTED***") {
		t.Errorf("expected ***REDACTED*** in diagnostic payload")
	}
}
