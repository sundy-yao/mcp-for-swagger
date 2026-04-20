package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sundy-yao/mcp-for-swagger/internal/logger"
	"github.com/sundy-yao/mcp-for-swagger/internal/mcp/registry"
	"github.com/sundy-yao/mcp-for-swagger/internal/mcp/transport"
	"github.com/sundy-yao/mcp-for-swagger/internal/mcp/types"
)

// MCPServer MCP 服务器
type MCPServer struct {
	name         string
	version      string
	instructions string
	registry     *registry.ToolRegistry
	host         string
	port         int
	transport    *transport.SSETransport
	mux          *http.ServeMux
}

// NewMCPServer 创建新的 MCP 服务器
func NewMCPServer(name, version, instructions, host string, port int) *MCPServer {
	server := &MCPServer{
		name:         name,
		version:      version,
		instructions: instructions,
		registry:     registry.GetGlobalRegistry(),
		host:         host,
		port:         port,
		transport:    transport.NewSSETransport(),
		mux:          http.NewServeMux(),
	}

	server.setupRoutes()
	return server
}

// setupRoutes 设置 HTTP 路由
func (s *MCPServer) setupRoutes() {
	s.mux.HandleFunc("/sse", s.logAndHandleSSE)
	s.mux.HandleFunc("/messages", s.logAndHandleMessage)
	s.mux.HandleFunc("/health", s.healthCheck)
}

// logAndHandleSSE 记录访问日志并调用 SSE 端点
func (s *MCPServer) logAndHandleSSE(w http.ResponseWriter, r *http.Request) {
	logger.Info("HTTP ACCESS: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	s.transport.SSEEndpoint(w, r)
}

// logAndHandleMessage 记录访问日志并处理消息
func (s *MCPServer) logAndHandleMessage(w http.ResponseWriter, r *http.Request) {
	logger.Info("HTTP ACCESS: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	s.handleMessage(w, r)
}

// handleMessage 处理 MCP 消息
func (s *MCPServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	logger.Debug("handleMessage received: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		logger.Warn("handleMessage: method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("handleMessage: failed to decode request: %v", err)
		s.sendError(w, nil, types.ParseError, fmt.Sprintf("Parse error: %v", err))
		return
	}

	logger.Debug("handleMessage: decoded request: method=%s, id=%v", req.Method, req.ID)

	ctx := r.Context()

	switch req.Method {
	case "initialize":
		s.handleInitialize(ctx, &req)
	case "notifications/initialized":
		s.handleInitialized(ctx, &req)
	case "tools/list":
		s.handleToolsList(ctx, &req)
	case "tools/call":
		s.handleToolsCall(ctx, &req)
	case "resources/list":
		s.handleResourcesList(ctx, &req)
	case "resources/read":
		s.handleResourcesRead(ctx, &req)
	case "prompts/list":
		s.handlePromptsList(ctx, &req)
	case "prompts/get":
		s.handlePromptsGet(ctx, &req)
	case "ping":
		s.handlePing(ctx, &req)
	default:
		logger.Warn("Unknown method: %s", req.Method)
		s.sendError(w, req.ID, types.MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}

	logger.Debug("handleMessage completed for method: %s", req.Method)

	// 返回 HTTP 200 OK
	body := []byte(`{"status":"ok"}`)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// handleInitialize 处理 initialize 请求
func (s *MCPServer) handleInitialize(ctx context.Context, req *types.JSONRPCRequest) {
	var params types.InitializeRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		logger.Error("Failed to parse initialize params: %v", err)
		return
	}

	// 协商协议版本
	negotiatedVersion := params.ProtocolVersion
	found := false
	for _, v := range types.SupportedProtocolVersions {
		if v == negotiatedVersion {
			found = true
			break
		}
	}
	if !found {
		negotiatedVersion = types.LatestProtocolVersion
	}

	logger.Info("Initialize received (id=%v), negotiated version: %s", req.ID, negotiatedVersion)

	response := types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: types.InitializeResponse{
			ProtocolVersion: negotiatedVersion,
			Capabilities: types.ServerCapabilities{
				Tools: make(map[string]interface{}),
			},
			ServerInfo: types.ServerInfo{
				Name:    s.name,
				Version: s.version,
			},
			Instructions: s.instructions,
		},
	}

	// 发送 initialize 响应
	responseData, _ := json.Marshal(response)
	var responseMap map[string]interface{}
	json.Unmarshal(responseData, &responseMap)
	if err := s.transport.SendToAllSessions(responseMap); err != nil {
		logger.Error("Failed to send initialize response: %v", err)
	}
	logger.Info("Sent initialize response via SSE")
}

// handleInitialized 处理 initialized 通知
func (s *MCPServer) handleInitialized(ctx context.Context, req *types.JSONRPCRequest) {
	logger.Info("Client initialized notification received, sending tools notification")

	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/tools/list_changed",
		"params":  map[string]interface{}{},
	}

	if err := s.transport.SendToAllSessions(notification); err != nil {
		logger.Error("Failed to send tools notification: %v", err)
	}
	logger.Info("Sent tools/list_changed notification via SSE")
}

// handleToolsList 处理 tools/list 请求
func (s *MCPServer) handleToolsList(ctx context.Context, req *types.JSONRPCRequest) {
	tools := s.registry.GetTools()
	logger.Info("tools/list called, returning %d tools", len(tools))

	// 调试：输出工具列表 JSON
	toolsJSON, _ := json.Marshal(tools)
	logger.Debug("Tools list JSON: %s", string(toolsJSON))

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"result": map[string]interface{}{
			"tools": tools,
		},
	}

	if err := s.transport.SendToAllSessions(response); err != nil {
		logger.Error("Failed to send tools/list response: %v", err)
	}
	logger.Info("Sent tools/list response via SSE")
}

// handleToolsCall 处理 tools/call 请求
func (s *MCPServer) handleToolsCall(ctx context.Context, req *types.JSONRPCRequest) {
	var params types.ToolsCallRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		logger.Error("handleToolsCall: invalid params: %v", err)
		s.sendErrorViaSSE(req.ID, types.InvalidParams, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	logger.Info("Calling tool: %s with args: %v", params.Name, params.Arguments)

	result, err := s.registry.CallTool(params.Name, params.Arguments)

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
	}

	if err != nil {
		logger.Error("Tool execution error: %v", err)
		response["error"] = map[string]interface{}{
			"code":    types.InternalError,
			"message": err.Error(),
		}
	} else {
		logger.Info("Tool %s executed successfully, result: %v", params.Name, result)
		// 将结果转换为 JSON 字符串
		resultJSON, _ := json.Marshal(result)
		response["result"] = map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": string(resultJSON)},
			},
		}
	}

	if err := s.transport.SendToAllSessions(response); err != nil {
		logger.Error("Failed to send tools/call response: %v", err)
	}
	logger.Info("Sent tools/call response via SSE")
}

// healthCheck 健康检查端点
func (s *MCPServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"sessions": s.transport.GetSessionCount(),
	})
}

// sendError 发送错误响应（HTTP）
func (s *MCPServer) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
	response := types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &types.ErrorInfo{Code: code, Message: message},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendErrorViaSSE 通过 SSE 发送错误响应
func (s *MCPServer) sendErrorViaSSE(id interface{}, code int, message string) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	s.transport.SendToAllSessions(response)
}

// handleResourcesList 处理 resources/list 请求
func (s *MCPServer) handleResourcesList(ctx context.Context, req *types.JSONRPCRequest) {
	logger.Info("resources/list called, no resources available")

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"result": map[string]interface{}{
			"resources": []interface{}{},
		},
	}

	if err := s.transport.SendToAllSessions(response); err != nil {
		logger.Error("Failed to send resources/list response: %v", err)
	}
	logger.Info("Sent resources/list response via SSE")
}

// handleResourcesRead 处理 resources/read 请求
func (s *MCPServer) handleResourcesRead(ctx context.Context, req *types.JSONRPCRequest) {
	logger.Info("resources/read called, no resources available")

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"error": map[string]interface{}{
			"code":    types.MethodNotFound,
			"message": "No resources supported",
		},
	}

	if err := s.transport.SendToAllSessions(response); err != nil {
		logger.Error("Failed to send resources/read response: %v", err)
	}
}

// handlePromptsList 处理 prompts/list 请求
func (s *MCPServer) handlePromptsList(ctx context.Context, req *types.JSONRPCRequest) {
	logger.Info("prompts/list called, no prompts available")

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"result": map[string]interface{}{
			"prompts": []interface{}{},
		},
	}

	if err := s.transport.SendToAllSessions(response); err != nil {
		logger.Error("Failed to send prompts/list response: %v", err)
	}
	logger.Info("Sent prompts/list response via SSE")
}

// handlePromptsGet 处理 prompts/get 请求
func (s *MCPServer) handlePromptsGet(ctx context.Context, req *types.JSONRPCRequest) {
	logger.Info("prompts/get called, no prompts available")

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"error": map[string]interface{}{
			"code":    types.MethodNotFound,
			"message": "No prompts supported",
		},
	}

	if err := s.transport.SendToAllSessions(response); err != nil {
		logger.Error("Failed to send prompts/get response: %v", err)
	}
}

// handlePing 处理 ping 请求
func (s *MCPServer) handlePing(ctx context.Context, req *types.JSONRPCRequest) {
	logger.Info("ping called")

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"result":  map[string]interface{}{},
	}

	if err := s.transport.SendToAllSessions(response); err != nil {
		logger.Error("Failed to send ping response: %v", err)
	}
	logger.Info("Sent ping response via SSE")
}

// Run 运行服务器
func (s *MCPServer) Run(host string, port int) error {
	if host == "" {
		host = s.host
	}
	if port == 0 {
		port = s.port
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Info("Starting %s on %s", s.name, addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  0, // 无限制，SSE 需要长连接
		WriteTimeout: 0, // 无限制，SSE 需要长连接
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}

// GetTransport 获取传输层（用于测试）
func (s *MCPServer) GetTransport() *transport.SSETransport {
	return s.transport
}

// GetRegistry 获取工具注册表（用于测试）
func (s *MCPServer) GetRegistry() *registry.ToolRegistry {
	return s.registry
}
