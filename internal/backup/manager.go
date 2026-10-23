package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupData represents a full snapshot of CubeX system configuration.
type BackupData struct {
	Version    string          `json:"version"`
	Timestamp  int64           `json:"timestamp"`
	CreatedAt  string          `json:"created_at"`
	State      json.RawMessage `json:"state,omitempty"`
	IR         json.RawMessage `json:"ir,omitempty"`
	HostConfig json.RawMessage `json:"host_config,omitempty"`
	ROSConfig  json.RawMessage `json:"ros_config,omitempty"`
	Devices    json.RawMessage `json:"devices,omitempty"`
	KSpeeder   json.RawMessage `json:"kspeeder,omitempty"`
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

	// 1. Export IR & State
	for _, fn := range []string{"config.ir.json", "cubex-state.json", "state.json"} {
		stateFile := filepath.Join(dataDir, fn)
		if d, err := os.ReadFile(stateFile); err == nil && len(d) > 0 {
			b.IR = json.RawMessage(d)
			b.State = json.RawMessage(d)
			break
		}
	}

	// 2. Export Host Config
	hostFile := filepath.Join(dataDir, "host-config.json")
	if d, err := os.ReadFile(hostFile); err == nil {
		b.HostConfig = json.RawMessage(d)
	}

	// 3. Export ROS Config
	rosFile := filepath.Join(dataDir, "ros-config.json")
	if d, err := os.ReadFile(rosFile); err == nil {
		b.ROSConfig = json.RawMessage(d)
	}

	// 5. Export Devices
	devFile := filepath.Join(dataDir, "devices.json")
	if d, err := os.ReadFile(devFile); err == nil {
		b.Devices = json.RawMessage(d)
	}

	// 6. Export KSpeeder Config
	kFile := filepath.Join(dataDir, "kspeeder-config.json")
	if d, err := os.ReadFile(kFile); err == nil {
		b.KSpeeder = json.RawMessage(d)
	}

	return b, nil
}

// ExportDesensitizedBackup exports the configuration with credentials, tokens, and private keys masked.
func (m *Manager) ExportDesensitizedBackup(version string) (*BackupData, error) {
	b, err := m.ExportBackup(version)
	if err != nil {
		return nil, err
	}
	// Desensitize IR/State
	if len(b.IR) > 0 {
		var irMap map[string]any
		if json.Unmarshal(b.IR, &irMap) == nil {
			if subs, ok := irMap["subscriptions"].([]any); ok {
				for _, s := range subs {
					if sm, ok := s.(map[string]any); ok {
						if u, ok := sm["url"].(string); ok && len(u) > 12 {
							sm["url"] = u[:8] + "****************" + u[len(u)-4:]
						}
					}
				}
			}
			if proxies, ok := irMap["proxies"].([]any); ok {
				for _, p := range proxies {
					if pm, ok := p.(map[string]any); ok {
						if _, ok := pm["password"]; ok {
							pm["password"] = "******"
						}
						if _, ok := pm["uuid"]; ok {
							pm["uuid"] = "********-****-****-****-************"
						}
						if _, ok := pm["private_key"]; ok {
							pm["private_key"] = "********************************************"
						}
					}
				}
			}
			if maskedBytes, err := json.Marshal(irMap); err == nil {
				b.IR = json.RawMessage(maskedBytes)
				b.State = json.RawMessage(maskedBytes)
			}
		}
	}
	// Desensitize ROS Config
	if len(b.ROSConfig) > 0 {
		var rosMap map[string]any
		if json.Unmarshal(b.ROSConfig, &rosMap) == nil {
			if _, ok := rosMap["password"]; ok {
				rosMap["password"] = "******"
			}
			if maskedBytes, err := json.Marshal(rosMap); err == nil {
				b.ROSConfig = json.RawMessage(maskedBytes)
			}
		}
	}
	// Desensitize Host Config
	if len(b.HostConfig) > 0 {
		var hostMap map[string]any
		if json.Unmarshal(b.HostConfig, &hostMap) == nil {
			if _, ok := hostMap["password"]; ok {
				hostMap["password"] = "******"
			}
			if maskedBytes, err := json.Marshal(hostMap); err == nil {
				b.HostConfig = json.RawMessage(maskedBytes)
			}
		}
	}
	return b, nil
}

// ImportBackup parses and restores configuration states from a BackupData struct.
func (m *Manager) ImportBackup(b *BackupData) error {
	if b == nil || (len(b.IR) == 0 && len(b.State) == 0) {
		return fmt.Errorf("无效的备份文件：缺失核心配置状态")
	}

	dataDir := filepath.Join(m.rootDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("无法创建数据目录: %w", err)
	}

	// 1. Restore IR & State
	corePayload := b.IR
	if len(corePayload) == 0 {
		corePayload = b.State
	}

	if len(corePayload) > 0 {
		for _, fn := range []string{"config.ir.json", "cubex-state.json", "state.json"} {
			stateFile := filepath.Join(dataDir, fn)
			_ = os.WriteFile(stateFile, corePayload, 0644)
		}
	}

	// 2. Restore host-config.json if present
	if len(b.HostConfig) > 0 {
		hostFile := filepath.Join(dataDir, "host-config.json")
		_ = os.WriteFile(hostFile, b.HostConfig, 0644)
	}

	// 3. Restore ros-config.json if present
	if len(b.ROSConfig) > 0 {
		rosFile := filepath.Join(dataDir, "ros-config.json")
		_ = os.WriteFile(rosFile, b.ROSConfig, 0644)
	}

	// 5. Restore devices.json if present
	if len(b.Devices) > 0 {
		devFile := filepath.Join(dataDir, "devices.json")
		_ = os.WriteFile(devFile, b.Devices, 0644)
	}

	// 6. Restore kspeeder-config.json if present
	if len(b.KSpeeder) > 0 {
		kFile := filepath.Join(dataDir, "kspeeder-config.json")
		_ = os.WriteFile(kFile, b.KSpeeder, 0644)
	}

	return nil
}
