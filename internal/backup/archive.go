package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

const (
	// MaxBackupUploadBytes caps incoming compressed .tar.gz backup uploads at 16 MiB.
	MaxBackupUploadBytes = 16 << 20
	// maxBackupEntryBytes caps each individual JSON file inside the backup archive at 8 MiB.
	maxBackupEntryBytes = 8 << 20
	// maxBackupTotalUncompressedBytes caps total extracted payload size at 24 MiB to defeat gzip bombs.
	maxBackupTotalUncompressedBytes = 24 << 20
	// maxBackupEntries caps the total number of tar entries processed during import.
	maxBackupEntries = 64
)

// allowlistedConfigFiles enumerates core configuration files included in lightweight migration packages.
// Runtime databases (*.db, *.sqlite), logs (*.log), auth credentials, and volatile runtime caches are excluded.
var allowlistedConfigFiles = []string{
	"config.ir.json",
	"user_overrides.json",
	"dns_rewrite.json",
	"domain_filter.json",
	"ui_state.json",
	"qos-config.json",
	"devices.json",
	"gateway_mode.json",
	"host-config.json",
	"ros-config.json",
	"ai-config.json",
}

var allowlistedConfigSet = func() map[string]bool {
	m := make(map[string]bool, len(allowlistedConfigFiles))
	for _, f := range allowlistedConfigFiles {
		m[f] = true
	}
	return m
}()

// ExportConfigTarGz packs desensitized core configuration files from <rootDir>/data into a .tar.gz stream.
func ExportConfigTarGz(rootDir string, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("output writer is nil")
	}
	dataDir := filepath.Join(rootDir, "data")

	gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		return fmt.Errorf("init gzip writer: %w", err)
	}
	tw := tar.NewWriter(gw)

	now := time.Now()
	for _, name := range allowlistedConfigFiles {
		srcPath := filepath.Join(dataDir, name)
		raw, err := os.ReadFile(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			_ = tw.Close()
			_ = gw.Close()
			return fmt.Errorf("read %s: %w", name, err)
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}

		payload := desensitizeConfigFile(name, raw)
		hdr := &tar.Header{
			Name:     name,
			Mode:     0600,
			Size:     int64(len(payload)),
			ModTime:  now,
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			_ = tw.Close()
			_ = gw.Close()
			return fmt.Errorf("write tar header %s: %w", name, err)
		}
		if _, err := tw.Write(payload); err != nil {
			_ = tw.Close()
			_ = gw.Close()
			return fmt.Errorf("write tar body %s: %w", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		_ = gw.Close()
		return fmt.Errorf("close tar writer: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	return nil
}

func desensitizeConfigFile(name string, raw []byte) []byte {
	switch name {
	case "config.ir.json":
		var irMap map[string]any
		if json.Unmarshal(raw, &irMap) != nil {
			return raw
		}
		if netMap, ok := irMap["network"].(map[string]any); ok {
			if sec, exists := netMap["clash_secret"].(string); exists && sec != "" {
				netMap["clash_secret"] = "******"
			}
		}
		if subs, ok := irMap["subscriptions"].([]any); ok {
			for _, item := range subs {
				if sm, ok := item.(map[string]any); ok {
					if u, ok := sm["url"].(string); ok && len(u) > 12 {
						sm["url"] = u[:8] + "****************" + u[len(u)-4:]
					} else if ok && u != "" {
						sm["url"] = "******"
					}
				}
			}
		}
		if proxies, ok := irMap["proxies"].([]any); ok {
			for _, item := range proxies {
				pm, ok := item.(map[string]any)
				if !ok {
					continue
				}
				maskStringKeys(pm, "password", "uuid", "private_key", "psk", "token")
				if rawCfg, ok := pm["raw_config"].(map[string]any); ok {
					maskStringKeys(rawCfg, "password", "uuid", "private_key", "private-key", "psk", "token", "auth")
				}
			}
		}
		if out, err := json.MarshalIndent(irMap, "", "  "); err == nil {
			return out
		}
	case "host-config.json", "ros-config.json", "ai-config.json":
		var cfgMap map[string]any
		if json.Unmarshal(raw, &cfgMap) != nil {
			return raw
		}
		maskStringKeys(cfgMap, "password", "private_key", "private-key", "token", "secret", "api_key", "apiKey")
		if out, err := json.MarshalIndent(cfgMap, "", "  "); err == nil {
			return out
		}
	}
	return raw
}

func maskStringKeys(m map[string]any, keys ...string) {
	for _, k := range keys {
		if v, exists := m[k]; exists {
			if s, ok := v.(string); ok && s != "" {
				m[k] = "******"
			}
		}
	}
}

// validateArchiveEntryName ensures the tar header name is strictly safe against path traversal,
// absolute paths, backslashes, and forbidden runtime/database file types.
func validateArchiveEntryName(rawName string, isDir bool) (string, error) {
	if rawName == "" {
		return "", fmt.Errorf("empty archive entry name")
	}
	if strings.ContainsRune(rawName, '\x00') {
		return "", fmt.Errorf("archive entry %q contains null byte", rawName)
	}
	if strings.ContainsRune(rawName, '\\') {
		return "", fmt.Errorf("archive entry %q contains forbidden backslash", rawName)
	}
	if strings.HasPrefix(rawName, "/") {
		return "", fmt.Errorf("archive entry %q uses forbidden absolute path", rawName)
	}
	if len(rawName) >= 2 && rawName[1] == ':' {
		return "", fmt.Errorf("archive entry %q uses forbidden drive prefix", rawName)
	}

	trimmed := strings.TrimPrefix(rawName, "./")
	if isDir {
		trimmed = strings.TrimSuffix(trimmed, "/")
		if trimmed == "" || trimmed == "." || trimmed == "data" {
			return "", nil
		}
		return "", fmt.Errorf("unexpected directory %q in backup archive", rawName)
	}

	for _, seg := range strings.Split(trimmed, "/") {
		if seg == ".." {
			return "", fmt.Errorf("archive entry %q contains path traversal", rawName)
		}
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("archive entry %q resolves outside root", rawName)
	}
	cleaned = strings.TrimPrefix(cleaned, "data/")
	if strings.Contains(cleaned, "/") {
		return "", fmt.Errorf("archive entry %q has forbidden nested path", rawName)
	}

	lower := strings.ToLower(cleaned)
	if strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".sqlite") || strings.HasSuffix(lower, ".sqlite3") || strings.HasSuffix(lower, ".log") {
		return "", fmt.Errorf("archive entry %q is a forbidden runtime/database file", rawName)
	}
	if !allowlistedConfigSet[cleaned] {
		return "", fmt.Errorf("archive entry %q is not an allowlisted configuration file", rawName)
	}
	return cleaned, nil
}

// ImportConfigTarGz strictly validates a .tar.gz backup stream in memory (rejecting path traversal,
// symlinks, oversized payloads, and invalid JSON) and atomically writes restored config files to <rootDir>/data.
func ImportConfigTarGz(rootDir string, r io.Reader, maxCompressedBytes int64) ([]string, error) {
	if r == nil {
		return nil, fmt.Errorf("empty backup stream")
	}
	if maxCompressedBytes <= 0 {
		maxCompressedBytes = MaxBackupUploadBytes
	}

	limited := io.LimitReader(r, maxCompressedBytes+1)
	gz, err := gzip.NewReader(limited)
	if err != nil {
		return nil, fmt.Errorf("invalid gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	staged := make(map[string][]byte)
	var totalBytes int64
	entries := 0

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid tar stream: %w", err)
		}

		entries++
		if entries > maxBackupEntries {
			return nil, fmt.Errorf("backup archive exceeds max entries limit (%d)", maxBackupEntries)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if _, err := validateArchiveEntryName(hdr.Name, true); err != nil {
				return nil, err
			}
			continue
		case tar.TypeReg, tar.TypeRegA:
		case tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
			continue
		case tar.TypeSymlink, tar.TypeLink:
			return nil, fmt.Errorf("forbidden symlink or hardlink entry %q in backup archive", hdr.Name)
		default:
			return nil, fmt.Errorf("unsupported tar entry type %d for %q", hdr.Typeflag, hdr.Name)
		}

		targetName, err := validateArchiveEntryName(hdr.Name, false)
		if err != nil {
			return nil, err
		}
		if hdr.Size < 0 || hdr.Size > maxBackupEntryBytes {
			return nil, fmt.Errorf("archive entry %q exceeds size limit", hdr.Name)
		}

		payload, err := io.ReadAll(io.LimitReader(tr, maxBackupEntryBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read entry %q: %w", hdr.Name, err)
		}
		if int64(len(payload)) > maxBackupEntryBytes {
			return nil, fmt.Errorf("archive entry %q exceeds uncompressed limit", hdr.Name)
		}
		totalBytes += int64(len(payload))
		if totalBytes > maxBackupTotalUncompressedBytes {
			return nil, fmt.Errorf("backup archive exceeds total uncompressed limit")
		}

		if !json.Valid(payload) {
			return nil, fmt.Errorf("archive entry %q contains invalid JSON", targetName)
		}
		if targetName == "config.ir.json" {
			var ir schema.ConfigIR
			if err := json.Unmarshal(payload, &ir); err != nil {
				return nil, fmt.Errorf("invalid config.ir.json schema: %w", err)
			}
		}

		staged[targetName] = payload
	}

	if len(staged) == 0 {
		return nil, fmt.Errorf("backup archive contains no restorable configuration files")
	}

	dataDir := filepath.Join(rootDir, "data")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	restored := make([]string, 0, len(staged))
	for name := range staged {
		restored = append(restored, name)
	}
	sort.Strings(restored)

	for _, name := range restored {
		dest := filepath.Join(dataDir, name)
		if err := schema.WriteFileAtomicSecure(dest, staged[name]); err != nil {
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
	}

	return restored, nil
}
