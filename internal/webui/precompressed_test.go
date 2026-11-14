package webui

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func makeGzipBytes(content string) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write([]byte(content))
	_ = gw.Close()
	return buf.Bytes()
}

func TestPrecompressedFileServer(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html.gz": {
			Data: makeGzipBytes("<html><body>CubeX Dashboard</body></html>"),
		},
		"assets/app.js.gz": {
			Data: makeGzipBytes("console.log('CubeX App');"),
		},
		"favicon.ico": {
			Data: []byte("favicon-bytes"),
		},
	}

	srv := NewPrecompressedFileServer(mockFS)

	// Test 1: Gzip request for app.js
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Expected Content-Encoding: gzip, got %s", rec.Header().Get("Content-Encoding"))
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("Expected immutable cache control for assets, got %s", rec.Header().Get("Cache-Control"))
	}

	// Test 2: Non-gzip request fallback decompression for index.html
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("Content-Encoding") != "" {
		t.Fatalf("Expected no Content-Encoding for non-gzip client, got %s", rec2.Header().Get("Content-Encoding"))
	}
	body, _ := io.ReadAll(rec2.Body)
	if !bytes.Contains(body, []byte("CubeX Dashboard")) {
		t.Fatalf("Expected decompressed content, got %s", string(body))
	}

	// Test 3: Uncompressed file favicon.ico
	req3 := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec3 := httptest.NewRecorder()
	srv.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", rec3.Code)
	}
	if rec3.Body.String() != "favicon-bytes" {
		t.Fatalf("Expected favicon-bytes, got %s", rec3.Body.String())
	}

	// Test 4: SPA fallback for unknown route /overview
	req4 := httptest.NewRequest(http.MethodGet, "/overview", nil)
	req4.Header.Set("Accept-Encoding", "gzip")
	rec4 := httptest.NewRecorder()
	srv.ServeHTTP(rec4, req4)

	if rec4.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", rec4.Code)
	}
	if rec4.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Expected gzip encoding on fallback, got %s", rec4.Header().Get("Content-Encoding"))
	}
}
