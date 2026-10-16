package sub

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

// Center handles universal multi-client subscription generation.
type Center struct {
	irProvider func() *schema.ConfigIR
}

// NewCenter creates a new subscription center.
func NewCenter(provider func() *schema.ConfigIR) *Center {
	return &Center{irProvider: provider}
}

// HandleKaring generates a high-performance profile tailored for Karing (iOS / Android / macOS / Windows).
func (c *Center) HandleKaring(w http.ResponseWriter, r *http.Request) {
	ir := c.irProvider()
	data, err := compiler.BuildMihomo(ir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Subscription-Userinfo", "upload=1073741824; download=53687091200; total=107374182400; expire=1893456000")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// HandleSingBox generates sing-box v1.14+ client JSON configuration.
func (c *Center) HandleSingBox(w http.ResponseWriter, r *http.Request) {
	ir := c.irProvider()
	data, err := compiler.BuildSingBox(ir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Subscription-Userinfo", "upload=1073741824; download=53687091200; total=107374182400; expire=1893456000")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// HandleClash generates Clash Meta / Mihomo compatible YAML.
func (c *Center) HandleClash(w http.ResponseWriter, r *http.Request) {
	c.HandleKaring(w, r)
}

// HandleShadowrocket generates base64-encoded proxy link list.
func (c *Center) HandleShadowrocket(w http.ResponseWriter, r *http.Request) {
	ir := c.irProvider()
	var links []string

	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" {
			continue
		}
		// Convert standard node to URI (Shadowsocks / Trojan / VMess / VLESS)
		switch strings.ToLower(p.Type) {
		case "shadowsocks", "ss":
			userinfo := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", p.Cipher, p.Password)))
			links = append(links, fmt.Sprintf("ss://%s@%s:%d#%s", userinfo, p.Server, p.Port, p.Name))
		case "trojan":
			links = append(links, fmt.Sprintf("trojan://%s@%s:%d#%s", p.Password, p.Server, p.Port, p.Name))
		case "vless":
			links = append(links, fmt.Sprintf("vless://%s@%s:%d?encryption=none&security=tls&sni=%s#%s", p.UUID, p.Server, p.Port, p.SNI, p.Name))
		}
	}

	raw := strings.Join(links, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(encoded))
}
