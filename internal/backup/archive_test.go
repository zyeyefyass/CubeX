package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildTestTarGz(t *testing.T, entries []struct {
	name     string
	typeflag byte
	body     string
	linkname string
}) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		tf := e.typeflag
		if tf == 0 {
			tf = tar.TypeReg
		}
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0600,
			Size:     int64(len(e.body)),
			Typeflag: tf,
			Linkname: e.linkname,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if tf == tar.TypeReg && len(e.body) > 0 {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExportAndImportConfigTarGz_DesensitizedAndFiltered(t *testing.T) {
	srcRoot := t.TempDir()
	srcData := filepath.Join(srcRoot, "data")
	if err := os.MkdirAll(srcData, 0700); err != nil {
		t.Fatal(err)
	}

	// Core config files
	irPayload := `{
		"meta": {"active_engine": "mihomo", "proxy_mode": "rule"},
		"network": {"clash_api_port": 9090, "clash_secret": "super-secret-token"},
		"subscriptions": [{"id": "s1", "name": "Airport", "url": "https://sub.example.com/api/v1/client?token=mysecret99"}],
		"proxies": [{"name": "HK-1", "type": "vless", "server": "hk.example.com", "port": 443, "uuid": "11111111-2222-3333-4444-555555555555", "password": "pw123"}]
	}`
	if err := os.WriteFile(filepath.Join(srcData, "config.ir.json"), []byte(irPayload), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcData, "dns_rewrite.json"), []byte(`{"enabled":true,"rules":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcData, "user_overrides.json"), []byte(`{"force_direct_domains":["internal.lan"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	// Runtime / DB / Log / Auth files that MUST NOT be exported
	for _, forbidden := range []string{"runtime.db", "cache.sqlite", "cubex.log", "console-auth.json", "traffic_state.json"} {
		if err := os.WriteFile(filepath.Join(srcData, forbidden), []byte("sensitive-runtime"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	var archive bytes.Buffer
	if err := ExportConfigTarGz(srcRoot, &archive); err != nil {
		t.Fatalf("ExportConfigTarGz failed: %v", err)
	}

	// Inspect tar contents directly to verify desensitization and exclusion
	gr, err := gzip.NewReader(bytes.NewReader(archive.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	exported := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		b := make([]byte, hdr.Size)
		_, _ = tr.Read(b)
		exported[hdr.Name] = string(b)
	}

	for _, forbidden := range []string{"runtime.db", "cache.sqlite", "cubex.log", "console-auth.json", "traffic_state.json"} {
		if _, ok := exported[forbidden]; ok {
			t.Fatalf("forbidden runtime/db/log file %q was exported!", forbidden)
		}
	}
	irExported := exported["config.ir.json"]
	if strings.Contains(irExported, "super-secret-token") || strings.Contains(irExported, "mysecret99") ||
		strings.Contains(irExported, "11111111-2222-3333-4444-555555555555") || strings.Contains(irExported, "pw123") {
		t.Fatalf("exported config.ir.json leaked unmasked secrets: %s", irExported)
	}

	// Restore into a fresh destination directory
	dstRoot := t.TempDir()
	restored, err := ImportConfigTarGz(dstRoot, bytes.NewReader(archive.Bytes()), MaxBackupUploadBytes)
	if err != nil {
		t.Fatalf("ImportConfigTarGz failed: %v", err)
	}
	if len(restored) != 3 {
		t.Fatalf("expected 3 restored items, got %v", restored)
	}
	rawRestored, err := os.ReadFile(filepath.Join(dstRoot, "data", "user_overrides.json"))
	if err != nil || !json.Valid(rawRestored) {
		t.Fatalf("restored user_overrides.json invalid or missing: %v", err)
	}
}

func TestImportConfigTarGz_SecurityRejections(t *testing.T) {
	cases := []struct {
		name    string
		entries []struct {
			name     string
			typeflag byte
			body     string
			linkname string
		}
	}{
		{
			name: "path_traversal",
			entries: []struct {
				name     string
				typeflag byte
				body     string
				linkname string
			}{
				{name: "ui_state.json", body: `{"ok":true}`},
				{name: "../../etc/passwd", body: `{"evil":true}`},
			},
		},
		{
			name: "absolute_path",
			entries: []struct {
				name     string
				typeflag byte
				body     string
				linkname string
			}{
				{name: "/etc/config.ir.json", body: `{}`},
			},
		},
		{
			name: "symlink",
			entries: []struct {
				name     string
				typeflag byte
				body     string
				linkname string
			}{
				{name: "config.ir.json", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
			},
		},
		{
			name: "invalid_json",
			entries: []struct {
				name     string
				typeflag byte
				body     string
				linkname string
			}{
				{name: "ui_state.json", body: `{"ok":true}`},
				{name: "dns_rewrite.json", body: `{broken json`},
			},
		},
		{
			name: "forbidden_db_file",
			entries: []struct {
				name     string
				typeflag byte
				body     string
				linkname string
			}{
				{name: "evil.db", body: `{}`},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dst := t.TempDir()
			raw := buildTestTarGz(t, tc.entries)
			if _, err := ImportConfigTarGz(dst, bytes.NewReader(raw), MaxBackupUploadBytes); err == nil {
				t.Fatalf("expected security/validation rejection for %s, got nil", tc.name)
			}
			// Ensure atomic validation did not leave partial writes
			if _, err := os.Stat(filepath.Join(dst, "data", "ui_state.json")); !os.IsNotExist(err) {
				t.Fatalf("expected atomic rejection with zero disk writes for %s", tc.name)
			}
		})
	}
}
