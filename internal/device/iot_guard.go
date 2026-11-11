package device

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// IoTDeviceCategory 智能设备类别
type IoTDeviceCategory string

const (
	CategoryCamera    IoTDeviceCategory = "智能监控摄像头"
	CategoryTVBox     IoTDeviceCategory = "电视盒子/投屏"
	CategorySpeaker   IoTDeviceCategory = "智能音箱/语音"
	CategoryGateway   IoTDeviceCategory = "智能家居中枢网关"
	CategoryAppliance IoTDeviceCategory = "智能家电/扫地机"
	CategoryGeneral   IoTDeviceCategory = "通用IoT终端"
)

// IoTDeviceRecord 智能设备沙盒画像记录
type IoTDeviceRecord struct {
	MAC                      string            `json:"mac"`
	IP                       string            `json:"ip"`
	Hostname                 string            `json:"hostname"`
	Vendor                   string            `json:"vendor"`
	Category                 IoTDeviceCategory `json:"category"`
	Sandboxed                bool              `json:"sandboxed"`
	UpLimitKbps              int               `json:"up_limit_kbps"` // 上行限速软上限 (Kbps)，0 = 默认 512Kbps 削峰
	CurrentUpSpeedKbps       float64           `json:"current_up_speed_kbps"`
	AbnormalBroadcastsCount  int64             `json:"abnormal_broadcasts_count"`
	ClippedEventsCount       int64             `json:"clipped_events_count"`
	LastActiveAt             time.Time         `json:"last_active_at"`
}

// IoTGuardStatus 遥测数据总览
type IoTGuardStatus struct {
	Enabled                         bool              `json:"enabled"`
	TotalMonitoredDevices           int               `json:"total_monitored_devices"`
	SandboxedDevicesCount           int               `json:"sandboxed_devices_count"`
	AbnormalBroadcastsIntercepted   int64             `json:"abnormal_broadcasts_intercepted"`
	P2PUploadPeaksClipped           int64             `json:"p2p_upload_peaks_clipped"`
	BandwidthSavedMB                float64           `json:"bandwidth_saved_mb"`
	SandboxedDevices                []IoTDeviceRecord `json:"sandboxed_devices"`
	UpdatedAt                       string            `json:"updated_at"`
	Summary                         string            `json:"summary"`
}

// IoTBehaviorGuard 全屋智能设备行为沙盒与上传防偷跑削峰管理器
type IoTBehaviorGuard struct {
	mu                      sync.RWMutex
	storePath               string
	enabled                 bool
	devices                 map[string]*IoTDeviceRecord
	totalIntercepted        atomic.Int64
	totalClipped            atomic.Int64
	bandwidthSavedMB        atomic.Int64
	lastUpdated             time.Time
}

var (
	defaultIoTGuard  *IoTBehaviorGuard
	onceIoTGuard    sync.Once
)

// GetIoTBehaviorGuard 获取单例
func GetIoTBehaviorGuard(dataDir ...string) *IoTBehaviorGuard {
	onceIoTGuard.Do(func() {
		dir := "data"
		if len(dataDir) > 0 && dataDir[0] != "" {
			dir = dataDir[0]
		}
		defaultIoTGuard = NewIoTBehaviorGuard(filepath.Join(dir, "iot_guard.json"))
	})
	return defaultIoTGuard
}

// NewIoTBehaviorGuard 实例化智能设备沙盒守卫
func NewIoTBehaviorGuard(storePath string) *IoTBehaviorGuard {
	guard := &IoTBehaviorGuard{
		storePath:   storePath,
		enabled:     true,
		devices:     make(map[string]*IoTDeviceRecord),
		lastUpdated: time.Now(),
	}

	_ = guard.loadPersisted()
	return guard
}

// GetStatus 获取当前遥测与沙盒状态
func (g *IoTBehaviorGuard) GetStatus() IoTGuardStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.getStatusLocked()
}

func (g *IoTBehaviorGuard) getStatusLocked() IoTGuardStatus {
	devs := make([]IoTDeviceRecord, 0, len(g.devices))
	sandboxedCount := 0
	for _, d := range g.devices {
		devs = append(devs, *d)
		if d.Sandboxed {
			sandboxedCount++
		}
	}

	return IoTGuardStatus{
		Enabled:                       g.enabled,
		TotalMonitoredDevices:         len(g.devices),
		SandboxedDevicesCount:         sandboxedCount,
		AbnormalBroadcastsIntercepted: g.totalIntercepted.Load(),
		P2PUploadPeaksClipped:         g.totalClipped.Load(),
		BandwidthSavedMB:              float64(g.bandwidthSavedMB.Load()),
		SandboxedDevices:              devs,
		UpdatedAt:                     g.lastUpdated.Format("15:04:05"),
		Summary:                       "全屋智能沙盒巡航中 · 拦截非授权广播 · 杜绝后台偷跑带宽",
	}
}

// ClassifyDevice 根据主机名与 MAC 前缀智能识别设备类别
func (g *IoTBehaviorGuard) ClassifyDevice(hostname, mac string) (IoTDeviceCategory, string) {
	hn := strings.ToLower(hostname)
	macUpper := strings.ToUpper(mac)

	if strings.Contains(hn, "cam") || strings.Contains(hn, "camera") || strings.Contains(hn, "ipc") {
		return CategoryCamera, "智能摄像头"
	}
	if strings.Contains(hn, "tv") || strings.Contains(hn, "box") || strings.Contains(hn, "roku") || strings.Contains(hn, "appletv") {
		return CategoryTVBox, "电视影音终端"
	}
	if strings.Contains(hn, "speaker") || strings.Contains(hn, "sound") || strings.Contains(hn, "sonos") || strings.Contains(hn, "tmall") {
		return CategorySpeaker, "智能音箱"
	}
	if strings.Contains(hn, "gateway") || strings.Contains(hn, "hub") || strings.Contains(hn, "zigbee") || strings.Contains(hn, "tuya") {
		return CategoryGateway, "智能中枢网关"
	}
	if strings.Contains(hn, "vacuum") || strings.Contains(hn, "clean") || strings.Contains(hn, "roborock") || strings.Contains(hn, "plug") {
		return CategoryAppliance, "智能家电"
	}

	// 厂商 OUI 匹配
	if strings.HasPrefix(macUpper, "64:90:C1") || strings.HasPrefix(macUpper, "7C:49:EB") {
		return CategoryCamera, "小米智能硬件"
	}
	if strings.HasPrefix(macUpper, "D8:0D:17") {
		return CategorySpeaker, "天猫精灵"
	}
	if strings.HasPrefix(macUpper, "50:EC:50") {
		return CategoryAppliance, "石头科技"
	}

	return CategoryGeneral, "智能IoT终端"
}

// SyncDevice 登记或更新设备
func (g *IoTBehaviorGuard) SyncDevice(ip, mac, hostname string, upSpeedBps uint64) {
	if mac == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	rec, ok := g.devices[mac]
	if !ok {
		cat, vendor := g.ClassifyDevice(hostname, mac)
		rec = &IoTDeviceRecord{
			MAC:         mac,
			IP:          ip,
			Hostname:    hostname,
			Vendor:      vendor,
			Category:    cat,
			Sandboxed:   true, // 默认对识别到的 IoT 设备开启防偷跑沙盒
			UpLimitKbps: 384,
		}
		g.devices[mac] = rec
	}

	if ip != "" {
		rec.IP = ip
	}
	if hostname != "" {
		rec.Hostname = hostname
	}
	rec.CurrentUpSpeedKbps = float64(upSpeedBps) / 1024.0 * 8.0
	rec.LastActiveAt = time.Now()
	g.lastUpdated = time.Now()
}

// SetDeviceSandbox 配置设备沙盒
func (g *IoTBehaviorGuard) SetDeviceSandbox(mac string, sandboxed bool, upLimitKbps int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	rec, ok := g.devices[mac]
	if !ok {
		rec = &IoTDeviceRecord{
			MAC:       mac,
			Sandboxed: sandboxed,
		}
		g.devices[mac] = rec
	}

	rec.Sandboxed = sandboxed
	if upLimitKbps >= 0 {
		rec.UpLimitKbps = upLimitKbps
	}
	rec.LastActiveAt = time.Now()
	g.lastUpdated = time.Now()

	_ = g.persist()
	return nil
}

// InspectAndClip 检查并实施上传突发削峰
func (g *IoTBehaviorGuard) InspectAndClip(mac string, currentUpSpeedBps uint64) (clipped bool, allowedSpeedBps uint64) {
	if !g.enabled {
		return false, currentUpSpeedBps
	}

	g.mu.RLock()
	rec, ok := g.devices[mac]
	g.mu.RUnlock()

	if !ok || !rec.Sandboxed {
		return false, currentUpSpeedBps
	}

	limitBps := uint64(rec.UpLimitKbps) * 1024 / 8
	if limitBps == 0 {
		limitBps = 384 * 1024 / 8 // 默认 384 Kbps
	}

	if currentUpSpeedBps > limitBps {
		savedBytes := currentUpSpeedBps - limitBps
		g.totalClipped.Add(1)
		g.bandwidthSavedMB.Add(int64(savedBytes / (1024 * 1024)))
		rec.ClippedEventsCount++
		return true, limitBps
	}

	return false, currentUpSpeedBps
}

// RecordAbnormalBroadcast 记录拦截非授权多播/广播
func (g *IoTBehaviorGuard) RecordAbnormalBroadcast(mac string) {
	g.totalIntercepted.Add(1)
	g.mu.Lock()
	if rec, ok := g.devices[mac]; ok {
		rec.AbnormalBroadcastsCount++
	}
	g.mu.Unlock()
}

// Toggle 启闭沙盒守卫
func (g *IoTBehaviorGuard) Toggle(enabled bool) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = enabled
	g.lastUpdated = time.Now()
	_ = g.persist()
	return g.enabled
}

func (g *IoTBehaviorGuard) persist() error {
	if g.storePath == "" {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(g.storePath), 0755)
	status := g.getStatusLocked()
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	tmp := g.storePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, g.storePath)
}

func (g *IoTBehaviorGuard) loadPersisted() error {
	if g.storePath == "" {
		return nil
	}
	data, err := os.ReadFile(g.storePath)
	if err != nil {
		return err
	}
	var st IoTGuardStatus
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}
	g.enabled = st.Enabled
	g.totalIntercepted.Store(st.AbnormalBroadcastsIntercepted)
	g.totalClipped.Store(st.P2PUploadPeaksClipped)
	g.bandwidthSavedMB.Store(int64(st.BandwidthSavedMB))
	for _, d := range st.SandboxedDevices {
		copyD := d
		g.devices[d.MAC] = &copyD
	}
	return nil
}
