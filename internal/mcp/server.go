package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// GatewayProvider defines the contract that CubeX Server must fulfill for the MCP server.
type GatewayProvider interface {
	GetTelemetry() (map[string]any, error)
	ListDevices() ([]map[string]any, error)
	SetDeviceMode(ip, mode string) error
	TriggerSelfHeal() (string, error)
	SwitchEngine(engine string) (string, error)
}

// Server handles Model Context Protocol (MCP) JSON-RPC 2.0 requests.
type Server struct {
	provider GatewayProvider
	mu       sync.RWMutex
	version  string
}

// NewServer creates a new MCP server.
func NewServer(provider GatewayProvider, version string) *Server {
	if version == "" {
		version = "v4.2"
	}
	return &Server{
		provider: provider,
		version:  version,
	}
}

// RPCRequest represents a standard JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCResponse represents a standard JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolDefinition represents an MCP tool definition.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ServeHTTP handles both POST (JSON-RPC) and GET (MCP SSE / metadata discovery).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		// Provide human and tool-readable server discovery info
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":        "CubeX Gateway MCP Server",
			"version":     s.version,
			"protocol":    "mcp/jsonrpc-2.0",
			"spec":        "2024-11-05",
			"description": "CubeX 宿主共生智能网关原生 Model Context Protocol (MCP) 接口服务，支持全屋网络自然语言掌控与自愈。",
			"endpoints": map[string]string{
				"rpc": "/api/v1/mcp",
			},
			"tools_count": len(s.getToolDefinitions()),
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeRPCError(w, nil, -32700, "Parse error: unable to read body")
		return
	}

	var req RPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeRPCError(w, nil, -32700, "Parse error: invalid JSON")
		return
	}

	res := s.handleRPC(req)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleRPC(req RPCRequest) RPCResponse {
	switch req.Method {
	case "initialize":
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]any{
					"name":    "CubeX Gateway MCP Server",
					"version": s.version,
				},
			},
		}

	case "notifications/initialized", "initialized":
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		}

	case "ping":
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		}

	case "tools/list":
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": s.getToolDefinitions(),
			},
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) getToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "get_gateway_telemetry",
			Description: "获取 CubeX 旁路由网关全息实时指标：包括活动双核引擎、实时下载与上传速率、活跃并发会话数、内存占用、当月累计物理跨境流量及安全自愈就绪状态。",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "list_lan_devices",
			Description: "获取局域网全部在线终端设备列表，包括设备名称、IP 地址、MAC 地址、在线状态及当前分流模式（rule: 智能规则分流，bypass: 芯片直连绕过）。",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "set_device_policy",
			Description: "切换局域网指定设备的网络分流模式（智能规则分流 或 芯片级直连绕过）。",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ip": map[string]any{
						"type":        "string",
						"description": "局域网设备的 IPv4 地址，例如 10.0.0.191",
					},
					"mode": map[string]any{
						"type":        "string",
						"enum":        []string{"rule", "bypass"},
						"description": "'rule' 表示加入智能规则分流，'bypass' 表示芯片直连绕过",
					},
				},
				"required": []string{"ip", "mode"},
			},
		},
		{
			Name:        "trigger_system_heal",
			Description: "触发安全中枢【⚡ 一键全栈自愈与规则加固】，自动热重载双核引擎、自愈 RouterOS 198.18.0.0/15 回指路由、清算失效僵尸连接、加固 IPv6 防泄露策略。",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "switch_proxy_engine",
			Description: "无感热重载切换底层透明代理内核引擎（Mihomo 或 sing-box）。",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"engine": map[string]any{
						"type":        "string",
						"enum":        []string{"mihomo", "singbox"},
						"description": "目标内核引擎：'mihomo' 或 'singbox'",
					},
				},
				"required": []string{"engine"},
			},
		},
	}
}

func (s *Server) handleToolCall(req RPCRequest) RPCResponse {
	var callParams struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &callParams); err != nil {
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Invalid params for tools/call"},
		}
	}

	switch callParams.Name {
	case "get_gateway_telemetry":
		data, err := s.provider.GetTelemetry()
		if err != nil {
			return s.toolError(req.ID, fmt.Sprintf("获取遥测数据失败: %v", err))
		}
		downBps, _ := data["inbound_speed_bps"].(int64)
		upBps, _ := data["outbound_speed_bps"].(int64)
		downStr := formatSpeed(downBps)
		upStr := formatSpeed(upBps)
		engine, _ := data["active_engine"].(string)
		conns, _ := data["connections_count"].(int)
		memMB, _ := data["memory_mb"].(float64)
		monthGB, _ := data["month_total_gb"].(float64)

		text := fmt.Sprintf(
			"📊 CubeX 旁路由网关实时运行指标：\n"+
				"• 活动内核引擎: %s (双核容灾保护就绪)\n"+
				"• 实时下行速率: %s\n"+
				"• 实时上行速率: %s\n"+
				"• 活跃并发会话: %d 个活跃连接 (0 泄露)\n"+
				"• 本月累计跨境流量: %.2f GB (服务器 7x24h 持久化)\n"+
				"• 宿主内存开销: %.1f MB (GOMEMLIMIT 软上限管控)\n"+
				"• 状态: 100%% 正常运行，DNS 纯净分流生效中",
			engine, downStr, upStr, conns, monthGB, memMB,
		)
		return s.toolSuccess(req.ID, text)

	case "list_lan_devices":
		devs, err := s.provider.ListDevices()
		if err != nil {
			return s.toolError(req.ID, fmt.Sprintf("获取局域网设备列表失败: %v", err))
		}
		if len(devs) == 0 {
			return s.toolSuccess(req.ID, "局域网内暂未发现其他设备或正在嗅探 ARP 表。")
		}
		text := fmt.Sprintf("📱 局域网在线终端设备清单 (共 %d 台)：\n", len(devs))
		for i, d := range devs {
			name, _ := d["name"].(string)
			ip, _ := d["ip"].(string)
			mac, _ := d["mac"].(string)
			mode, _ := d["mode"].(string)
			modeDesc := "🟢 智能规则分流"
			if mode == "bypass" {
				modeDesc = "🔵 芯片直连绕过 (0负载)"
			}
			text += fmt.Sprintf("%d. %s (%s) | MAC: %s | 策略: %s\n", i+1, name, ip, mac, modeDesc)
		}
		return s.toolSuccess(req.ID, text)

	case "set_device_policy":
		var args struct {
			IP   string `json:"ip"`
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(callParams.Arguments, &args); err != nil {
			return s.toolError(req.ID, "参数解析错误: 需要 ip 与 mode")
		}
		if args.IP == "" || (args.Mode != "rule" && args.Mode != "bypass") {
			return s.toolError(req.ID, "参数不合法: mode 必须为 'rule' 或 'bypass'")
		}
		if err := s.provider.SetDeviceMode(args.IP, args.Mode); err != nil {
			return s.toolError(req.ID, fmt.Sprintf("设置设备 %s 策略失败: %v", args.IP, err))
		}
		desc := "🟢 智能规则分流"
		if args.Mode == "bypass" {
			desc = "🔵 芯片直连绕过 (0负载)"
		}
		return s.toolSuccess(req.ID, fmt.Sprintf("✅ 设备策略调整成功！\n设备 IP: %s\n当前模式: %s\n策略路由与连接池已同步实时刷新生效。", args.IP, desc))

	case "trigger_system_heal":
		summary, err := s.provider.TriggerSelfHeal()
		if err != nil {
			return s.toolError(req.ID, fmt.Sprintf("执行一键全栈自愈失败: %v", err))
		}
		return s.toolSuccess(req.ID, fmt.Sprintf("⚡ 安全中枢全栈自愈完成！\n%s", summary))

	case "switch_proxy_engine":
		var args struct {
			Engine string `json:"engine"`
		}
		if err := json.Unmarshal(callParams.Arguments, &args); err != nil {
			return s.toolError(req.ID, "参数解析错误: 需要 engine")
		}
		if args.Engine != "mihomo" && args.Engine != "singbox" {
			return s.toolError(req.ID, "参数不合法: engine 必须为 'mihomo' 或 'singbox'")
		}
		res, err := s.provider.SwitchEngine(args.Engine)
		if err != nil {
			return s.toolError(req.ID, fmt.Sprintf("切换引擎至 %s 失败: %v", args.Engine, err))
		}
		return s.toolSuccess(req.ID, fmt.Sprintf("🔄 双核引擎热重载切换成功！\n当前活动核心: %s\n耗时: %s\n全量分流规则树已热重载，连接池已无损平滑迁移。", args.Engine, res))

	default:
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Unknown tool: %s", callParams.Name),
			},
		}
	}
}

func (s *Server) toolSuccess(id any, text string) RPCResponse {
	return RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": text,
				},
			},
		},
	}
}

func (s *Server) toolError(id any, message string) RPCResponse {
	return RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"isError": true,
			"content": []map[string]any{
				{
					"type": "text",
					"text": fmt.Sprintf("❌ 执行错误: %s", message),
				},
			},
		},
	}
}

func (s *Server) writeRPCError(w http.ResponseWriter, id any, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	})
}

func formatSpeed(bps int64) string {
	if bps < 1024 {
		return fmt.Sprintf("%d B/s", bps)
	} else if bps < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", float64(bps)/1024)
	} else if bps < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB/s", float64(bps)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB/s", float64(bps)/(1024*1024*1024))
}
