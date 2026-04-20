package types

import "encoding/json"

// 协议版本常量
const (
	LatestProtocolVersion = "2024-11-05"
)

var SupportedProtocolVersions = []string{
	"2024-11-05",
	"2025-03-26",
	"2025-06-18",
}

// JSON-RPC 错误码
const (
	ParseError    = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// ToolDefinition MCP 工具定义
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler 工具处理函数类型
type ToolHandler func(args map[string]interface{}) (interface{}, error)

// ServerInfo 服务器信息
type ServerInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Instructions string `json:"instructions,omitempty"`
}

// ServerCapabilities 服务器能力
type ServerCapabilities struct {
	Tools     map[string]interface{} `json:"tools,omitempty"`
	Resources map[string]interface{} `json:"resources,omitempty"`
	Prompts   map[string]interface{} `json:"prompts,omitempty"`
}

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo JSON-RPC 错误信息
type ErrorInfo struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeRequest 初始化请求参数
type InitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion,omitempty"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
	ClientInfo      ServerInfo             `json:"clientInfo,omitempty"`
}

// InitializeResponse 初始化响应参数
type InitializeResponse struct {
	ProtocolVersion string              `json:"protocolVersion"`
	Capabilities    ServerCapabilities  `json:"capabilities"`
	ServerInfo      ServerInfo          `json:"serverInfo"`
	Instructions    string              `json:"instructions,omitempty"`
}

// ToolsCallRequest 工具调用请求
type ToolsCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ContentItem 内容项
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ResourceDefinition MCP 资源定义
type ResourceDefinition struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
}

// PromptDefinition MCP Prompt 定义
type PromptDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
}

// PromptArgument Prompt 参数
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}
