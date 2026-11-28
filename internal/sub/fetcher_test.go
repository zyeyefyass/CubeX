package sub

import (
	"bytes"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetcherErrorAttribution(t *testing.T) {
	f := NewFetcher()

	t.Run("http_status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()
		_, err := f.doFetch(srv.URL+"/api/sub?token=secret123", f.client)
		cat, dom, _ := ClassifyFetchError(err)
		if cat != "http_status" {
			t.Fatalf("category = %q, want http_status (err=%v)", cat, err)
		}
		if dom != "127.0.0.1" {
			t.Fatalf("domain = %q, want 127.0.0.1", dom)
		}
		if strings.Contains(err.Error(), "secret123") {
			t.Fatalf("error leaked subscription query token: %v", err)
		}
	})

	t.Run("parse_failed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
		}))
		defer srv.Close()
		_, err := f.doFetch(srv.URL+"/sub?token=abc", f.client)
		cat, _, _ := ClassifyFetchError(err)
		if cat != "parse_failed" {
			t.Fatalf("category = %q, want parse_failed (err=%v)", cat, err)
		}
	})

	t.Run("body_too_large", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chunk := bytes.Repeat([]byte("A"), 1024*1024)
			for i := 0; i < 16; i++ {
				_, _ = w.Write(chunk)
			}
		}))
		defer srv.Close()
		_, err := f.doFetch(srv.URL, f.client)
		cat, _, _ := ClassifyFetchError(err)
		if cat != "body_too_large" {
			t.Fatalf("category = %q, want body_too_large (err=%v)", cat, err)
		}
	})

	t.Run("dns_failed", func(t *testing.T) {
		cat := classifyRawErrorString(&net.DNSError{Err: "no such host", Name: "bad.invalid"})
		if cat != "dns_failed" {
			t.Fatalf("category = %q, want dns_failed", cat)
		}
	})

	t.Run("connect_timeout", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(300 * time.Millisecond)
		}))
		defer srv.Close()
		shortClient := &http.Client{Timeout: 50 * time.Millisecond}
		_, err := f.doFetch(srv.URL+"?token=SECRET_TOK", shortClient)
		cat, _, _ := ClassifyFetchError(err)
		if cat != "connect_timeout" {
			t.Fatalf("category = %q, want connect_timeout (err=%v)", cat, err)
		}
		if strings.Contains(err.Error(), "SECRET_TOK") {
			t.Fatalf("expected error string desensitized without URL token, got %q", err.Error())
		}
	})

	t.Run("tls_failed", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer srv.Close()
		strictClient := &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: false}},
		}
		_, err := f.doFetch(srv.URL, strictClient)
		cat, _, _ := ClassifyFetchError(err)
		if cat != "tls_failed" {
			t.Fatalf("category = %q, want tls_failed (err=%v)", cat, err)
		}
		var fe *FetchError
		if !errors.As(err, &fe) {
			t.Fatalf("expected *FetchError")
		}
	})
}
