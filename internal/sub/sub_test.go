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
