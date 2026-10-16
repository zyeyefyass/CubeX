package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupData represents a full snapshot of MiCube system configuration.
type BackupData struct {
	Version   string          `json:"version"`
	Timestamp int64           `json:"timestamp"`
	CreatedAt string          `json:"created_at"`
	State     json.RawMessage `json:"state"`
	WireGuard json.RawMessage `json:"wireguard,omitempty"`
	ROSConfig json.RawMessage `json:"ros_config,omitempty"`
	Devices   json.RawMessage `json:"devices,omitempty"`
}

// Manager handles exporting and importing full system configuration.
type Manager struct {
	rootDir string
}

// NewManager creates a new backup Manager instance.
func NewManager(rootDir string) *Manager {
	return &Manager{rootDir: rootDir}
}

// ExportBackup packs all configuration states into a single BackupData struct.
func (m *Manager) ExportBackup(version string) (*BackupData, error) {
	dataDir := filepath.Join(m.rootDir, "data")

	b := &BackupData{
		Version:   version,
		Timestamp: time.Now().Unix(),
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	for _, fn := range []string{"micube-state.json", "singcube-state.json", "state.json"} {
		stateFile := filepath.Join(dataDir, fn)
		if d, err := os.ReadFile(stateFile); err == nil && len(d) > 0 {
			b.State = json.RawMessage(d)
			break
		}
	}

	wgFile := filepath.Join(dataDir, "wireguard.json")
	if d, err := os.ReadFile(wgFile); err == nil {
		b.WireGuard = json.RawMessage(d)
	}

	rosFile := filepath.Join(dataDir, "ros-config.json")
	if d, err := os.ReadFile(rosFile); err == nil {
		b.ROSConfig = json.RawMessage(d)
	}

	devFile := filepath.Join(dataDir, "devices.json")
	if d, err := os.ReadFile(devFile); err == nil {
		b.Devices = json.RawMessage(d)
	}

	return b, nil
}

// ImportBackup parses and restores configuration states from a BackupData struct.
func (m *Manager) ImportBackup(b *BackupData) error {
	if b == nil || len(b.State) == 0 {
		return fmt.Errorf("无效的备份文件：缺失核心配置状态")
	}

	dataDir := filepath.Join(m.rootDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	// 1. Restore state files across all recognized names for 100% universal inter-compatibility
	for _, fn := range []string{"micube-state.json", "singcube-state.json", "state.json"} {
		stateFile := filepath.Join(dataDir, fn)
		_ = os.WriteFile(stateFile, b.State, 0644)
	}

	// 2. Restore wireguard.json if present
	if len(b.WireGuard) > 0 {
		wgFile := filepath.Join(dataDir, "wireguard.json")
		_ = os.WriteFile(wgFile, b.WireGuard, 0644)
	}

	// 3. Restore ros-config.json if present
	if len(b.ROSConfig) > 0 {
		rosFile := filepath.Join(dataDir, "ros-config.json")
		_ = os.WriteFile(rosFile, b.ROSConfig, 0644)
	}

	// 4. Restore devices.json if present
	if len(b.Devices) > 0 {
		devFile := filepath.Join(dataDir, "devices.json")
		_ = os.WriteFile(devFile, b.Devices, 0644)
	}

	return nil
}
