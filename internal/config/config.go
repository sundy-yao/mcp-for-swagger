package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// OpenAPIConfig OpenAPI 规范配置
type OpenAPIConfig struct {
	Path       string   `yaml:"path"`        // OpenAPI YAML 文件路径
	URL        string   `yaml:"url"`         // 或者从 URL 加载
	BaseURL    string   `yaml:"base_url"`    // 后端 API 基础 URL
	AuthHeader string   `yaml:"auth_header"` // 认证头 (e.g., "X-Api-Token:xxx" 或 "Bearer xxx")，兼容旧格式
	Headers    []string `yaml:"headers"`     // 多个 HTTP headers，格式："Header-Name:value"
}

// ToolMapping 工具映射配置（可选，用于自定义）
type ToolMapping struct {
	Prefix      string            `yaml:"prefix"`       // 工具名前缀
	Exclude     []string          `yaml:"exclude"`      // 排除的端点
	IncludeTags []string          `yaml:"include_tags"` // 只包含特定 tag 的端点
	Custom      []CustomTool      `yaml:"custom"`       // 自定义工具
}

// CustomTool 自定义工具配置
type CustomTool struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	OperationID string            `yaml:"operation_id"` // 关联的 OpenAPI operationId
	Headers     map[string]string `yaml:"headers"`      // 额外的 headers
}

// MCPSettings MCP 服务器配置
type MCPSettings struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Name      string `yaml:"name"`
	Version   string `yaml:"version"`
	Transport string `yaml:"transport"` // sse 或 stdio
}

// LogSettings 日志配置
type LogSettings struct {
	Level         string `yaml:"level"`
	Dir           string `yaml:"dir"`
	File          string `yaml:"file"`
	RetentionDays int    `yaml:"retention_days"`
}

// Settings 配置结构
type Settings struct {
	MCP      MCPSettings  `yaml:"mcp"`
	OpenAPI  OpenAPIConfig `yaml:"openapi"`
	ToolMap  ToolMapping   `yaml:"tool_mapping,omitempty"`
	Log      LogSettings   `yaml:"log,omitempty"`
}

// LoadConfig 从 YAML 文件加载配置
func LoadConfig(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 应用环境变量覆盖
	settings.applyEnvOverrides()

	// 设置默认值
	settings.setDefaults()

	return &settings, nil
}

// applyEnvOverrides 应用环境变量覆盖
func (s *Settings) applyEnvOverrides() {
	if host := os.Getenv("MCP_HOST"); host != "" {
		s.MCP.Host = host
	}
	if port := os.Getenv("MCP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			s.MCP.Port = p
		}
	}
	if name := os.Getenv("MCP_NAME"); name != "" {
		s.MCP.Name = name
	}
	if transport := os.Getenv("MCP_TRANSPORT"); transport != "" {
		s.MCP.Transport = transport
	}
	if baseURL := os.Getenv("API_BASE_URL"); baseURL != "" {
		s.OpenAPI.BaseURL = baseURL
	}
	if authHeader := os.Getenv("API_AUTH_HEADER"); authHeader != "" {
		s.OpenAPI.AuthHeader = authHeader
	}
	if authToken := os.Getenv("API_AUTH_TOKEN"); authToken != "" {
		// 如果配置了 API_AUTH_TOKEN，自动构建 headers
		s.OpenAPI.Headers = []string{fmt.Sprintf("X-Api-Token:%s", authToken)}
	}

	// 处理配置中的 ${VAR} 占位符
	s.MCP.Host = replaceEnvVars(s.MCP.Host)
	s.OpenAPI.BaseURL = replaceEnvVars(s.OpenAPI.BaseURL)

	// 处理 headers 中的占位符
	for i, header := range s.OpenAPI.Headers {
		s.OpenAPI.Headers[i] = replaceEnvVars(header)
	}
	if s.OpenAPI.AuthHeader != "" {
		s.OpenAPI.AuthHeader = replaceEnvVars(s.OpenAPI.AuthHeader)
	}
}

// replaceEnvVars 替换字符串中的 ${VAR} 占位符
func replaceEnvVars(s string) string {
	if s == "" {
		return s
	}

	// 处理 ${VAR} 格式
	for {
		start := strings.Index(s, "${")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "}")
		if end == -1 {
			break
		}
		end += start

		varName := s[start+2 : end]
		varValue := os.Getenv(varName)

		s = s[:start] + varValue + s[end+1:]
	}

	return s
}

// setDefaults 设置默认值
func (s *Settings) setDefaults() {
	if s.MCP.Host == "" {
		s.MCP.Host = "0.0.0.0"
	}
	if s.MCP.Port == 0 {
		s.MCP.Port = 8000
	}
	if s.MCP.Name == "" {
		s.MCP.Name = "mcp-openapi-service"
	}
	if s.MCP.Version == "" {
		s.MCP.Version = "0.1.0"
	}
	if s.MCP.Transport == "" {
		s.MCP.Transport = "sse"
	}
	if s.Log.Level == "" {
		s.Log.Level = "INFO"
	}
	if s.Log.Dir == "" {
		s.Log.Dir = "./logs"
	}
	if s.Log.File == "" {
		s.Log.File = "app.log"
	}
	if s.Log.RetentionDays == 0 {
		s.Log.RetentionDays = 3
	}
}
