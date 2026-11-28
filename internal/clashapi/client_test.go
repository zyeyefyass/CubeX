package clashapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientNormalizesBaseURLAndAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path == "/connections" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(ConnectionsResponse{
				Connections: []ConnectionEntry{{ID: "c1"}},
			})
			return
		}
		if r.URL.Path == "/connections" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// 故意剥掉 http:// 前缀与追加斜杠，验证构造器能自动规范化
	bareHost := strings.TrimPrefix(srv.URL, "http://") + "/"
	c := NewClient(bareHost, "sec-123", nil)
	if !strings.HasPrefix(c.BaseURL, "http://") {
		t.Fatalf("expected http:// prefix, got %q", c.BaseURL)
	}
	conns, err := c.GetConnections(context.Background())
	if err != nil || len(conns.Connections) != 1 {
		t.Fatalf("GetConnections failed: %v, %+v", err, conns)
	}
	if gotAuth != "Bearer sec-123" {
		t.Fatalf("expected Bearer auth header, got %q", gotAuth)
	}
	if err := c.DeleteAllConnections(context.Background()); err != nil {
		t.Fatalf("DeleteAllConnections failed: %v", err)
	}
}
