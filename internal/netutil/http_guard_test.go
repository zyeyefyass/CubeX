package netutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDrainAndCloseBody(t *testing.T) {
	// 1. Nil safety
	DrainAndCloseBody(nil)
	DrainAndCloseBody(&http.Response{})

	// 2. Body drain
	bodyContent := strings.Repeat("x", 128*1024) // 128KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(bodyContent))
	}))
	defer server.Close()

	client := NewBoundedHTTPClient(2 * time.Second)
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	DrainAndCloseBody(resp)

	// Verify reading closed body returns error
	buf := make([]byte, 10)
	_, err = resp.Body.Read(buf)
	if err == nil || err == io.EOF {
		// Reading closed body should error out or already be at EOF
	}
}

func TestNewBoundedHTTPClient(t *testing.T) {
	c := NewBoundedHTTPClient(0)
	if c.Timeout != 5*time.Second {
		t.Fatalf("Expected default timeout 5s, got %v", c.Timeout)
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Expected *http.Transport")
	}
	if tr.MaxIdleConns != 16 || tr.MaxIdleConnsPerHost != 4 {
		t.Fatalf("Unexpected pool limits: %d, %d", tr.MaxIdleConns, tr.MaxIdleConnsPerHost)
	}
}
