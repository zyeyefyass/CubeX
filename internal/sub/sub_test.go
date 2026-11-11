package sub_test

import (
	"encoding/base64"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/sub"
)

func TestParseClashYAML(t *testing.T) {
	sampleClash := `
proxies:
  - name: "HK-01"
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-128-gcm
    password: mypassword
  - name: "JP-02"
    type: vless
    server: 5.6.7.8
    port: 443
    uuid: aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
    tls: true
    servername: jp.example.com
`
	nodes, err := sub.ParseSubscription([]byte(sampleClash))
	if err != nil {
		t.Fatalf("ParseSubscription Clash YAML failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "HK-01" || nodes[0].Type != "ss" {
		t.Errorf("Node 0 mismatch: %+v", nodes[0])
	}
	if nodes[1].Name != "JP-02" || nodes[1].SNI != "jp.example.com" {
		t.Errorf("Node 1 mismatch: %+v", nodes[1])
	}
}

func TestParseSingBoxJSON(t *testing.T) {
	sampleSingbox := `
{
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    },
    {
      "type": "trojan",
      "tag": "SG-Trojan-03",
      "server": "sg.example.com",
      "server_port": 443,
      "password": "secretpassword",
      "tls": {
        "enabled": true,
        "server_name": "sg.example.com"
      }
    }
  ]
}
`
	nodes, err := sub.ParseSubscription([]byte(sampleSingbox))
	if err != nil {
		t.Fatalf("ParseSubscription sing-box JSON failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node (excluding direct), got %d", len(nodes))
	}
	if nodes[0].Name != "SG-Trojan-03" || nodes[0].Type != "trojan" {
		t.Errorf("Node 0 mismatch: %+v", nodes[0])
	}
}

func TestParseBase64Links(t *testing.T) {
	rawLinks := "trojan://password123@us.example.com:443?sni=us.example.com#US-Trojan-01\n" +
		"vless://11111111-2222-3333-4444-555555555555@hk.example.com:443?security=tls&sni=hk.example.com#HK-Vless-02"
	encoded := base64.StdEncoding.EncodeToString([]byte(rawLinks))

	nodes, err := sub.ParseSubscription([]byte(encoded))
	if err != nil {
		t.Fatalf("ParseSubscription base64 failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "US-Trojan-01" {
		t.Errorf("Node 0 name mismatch: %s", nodes[0].Name)
	}
	if nodes[1].Name != "HK-Vless-02" {
		t.Errorf("Node 1 name mismatch: %s", nodes[1].Name)
	}
}

func TestParseHysteria2PortHopping(t *testing.T) {
	link := "hysteria2://mysecretpass@103.204.174.208:8443?mport=20000-40000&sni=www.yahoo.co.jp&insecure=1#JP-Hy2-Hopping"
	node, err := sub.ParseURILink(link)
	if err != nil {
		t.Fatalf("ParseURILink failed: %v", err)
	}
	if node.Type != "hysteria2" {
		t.Errorf("Expected hysteria2, got %s", node.Type)
	}
	if node.Port != 8443 {
		t.Errorf("Expected port 8443, got %d", node.Port)
	}
	if node.Ports != "20000-40000" {
		t.Errorf("Expected ports 20000-40000, got %s", node.Ports)
	}
	if node.RawConfig["mport"] != "20000-40000" {
		t.Errorf("Expected mport in raw config, got %v", node.RawConfig["mport"])
	}
}

func TestParseVlessRealityWithSpiderX(t *testing.T) {
	link := "vless://12345678-1234-1234-1234-1234567890ab@reality.example.com:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.apple.com&fp=chrome&pbk=fakePublicKey1234567890abcdefghijklmnop&sid=12345678&spx=%2Fnews#US-Reality-Node"
	node, err := sub.ParseURILink(link)
	if err != nil {
		t.Fatalf("ParseURILink reality failed: %v", err)
	}
	if node.Type != "vless" {
		t.Fatalf("Expected vless, got %s", node.Type)
	}
	raw := node.RawConfig
	if raw["client-fingerprint"] != "chrome" {
		t.Errorf("Expected client-fingerprint chrome, got %v", raw["client-fingerprint"])
	}
	realityOpts, ok := raw["reality-opts"].(map[string]any)
	if !ok {
		t.Fatalf("Expected reality-opts map, got %v", raw["reality-opts"])
	}
	if realityOpts["public-key"] != "fakePublicKey1234567890abcdefghijklmnop" {
		t.Errorf("Expected public-key match, got %v", realityOpts["public-key"])
	}
	if realityOpts["short-id"] != "12345678" {
		t.Errorf("Expected short-id 12345678, got %v", realityOpts["short-id"])
	}
	if realityOpts["spider-x"] != "/news" {
		t.Errorf("Expected spider-x /news, got %v", realityOpts["spider-x"])
	}
}

func TestUniversalSmartParse(t *testing.T) {
	markdownInput := "```yaml\nproxies:\n  - name: \"SG-01\"\n    type: ss\n    server: 1.1.1.1\n    port: 8388\n    cipher: aes-128-gcm\n    password: pass\n```"
	nodes, err := sub.UniversalSmartParse(markdownInput)
	if err != nil {
		t.Fatalf("UniversalSmartParse failed: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Name != "SG-01" {
		t.Errorf("Unexpected nodes: %+v", nodes)
	}
}

func TestParseSnellAndTransports(t *testing.T) {
	snellV6, err := sub.ParseURILink("snell://mypsk123@snell.example.com:6160?version=6&obfs=http&obfs-host=bing.com#SG-Snell-v6")
	if err != nil {
		t.Fatalf("ParseURILink snell v6 failed: %v", err)
	}
	if snellV6.Type != "snell" || snellV6.Password != "mypsk123" || snellV6.Port != 6160 {
		t.Errorf("Snell v6 node mismatch: %+v", snellV6)
	}
	if snellV6.RawConfig["version"] != 6 || snellV6.RawConfig["psk"] != "mypsk123" {
		t.Errorf("Snell v6 RawConfig mismatch: %+v", snellV6.RawConfig)
	}
	obfsOpts, _ := snellV6.RawConfig["obfs-opts"].(map[string]any)
	if obfsOpts["mode"] != "http" || obfsOpts["host"] != "bing.com" {
		t.Errorf("Snell v6 obfs-opts mismatch: %+v", obfsOpts)
	}

	huNode, err := sub.ParseURILink("vless://11111111-2222-3333-4444-555555555555@hu.example.com:443?security=tls&type=httpupgrade&path=%2Fhu&host=cdn.example.com#US-HttpUpgrade")
	if err != nil {
		t.Fatalf("ParseURILink httpupgrade failed: %v", err)
	}
	if huNode.Network != "httpupgrade" || huNode.Path != "/hu" {
		t.Errorf("HttpUpgrade mismatch: %+v", huNode)
	}
	huOpts, _ := huNode.RawConfig["httpupgrade-opts"].(map[string]any)
	if huOpts["path"] != "/hu" || huOpts["host"] != "cdn.example.com" {
		t.Errorf("HttpUpgrade opts mismatch: %+v", huOpts)
	}

	xhNode, err := sub.ParseURILink("vless://11111111-2222-3333-4444-555555555555@xh.example.com:443?security=tls&type=xhttp&path=%2Fxh&host=edge.example.com&mode=auto#JP-XHTTP")
	if err != nil {
		t.Fatalf("ParseURILink xhttp failed: %v", err)
	}
	if xhNode.Network != "xhttp" || xhNode.Path != "/xh" {
		t.Errorf("XHTTP mismatch: %+v", xhNode)
	}
	xhOpts, _ := xhNode.RawConfig["xhttp-opts"].(map[string]any)
	if xhOpts["path"] != "/xh" || xhOpts["host"] != "edge.example.com" || xhOpts["mode"] != "auto" {
		t.Errorf("XHTTP opts mismatch: %+v", xhOpts)
	}
}
