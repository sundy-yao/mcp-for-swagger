package openapi

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parameter OpenAPI 参数
type Parameter struct {
	Name        string                 `yaml:"name"`
	In          string                 `yaml:"in"` // query, path, header, cookie
	Required    bool                   `yaml:"required,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Schema      map[string]interface{} `yaml:"schema,omitempty"`
}

// RequestBody OpenAPI 请求体
type RequestBody struct {
	Description string                 `yaml:"description,omitempty"`
	Required    bool                   `yaml:"required,omitempty"`
	Content     map[string]interface{} `yaml:"content,omitempty"`
}

// Operation OpenAPI 操作
type Operation struct {
	OperationID string                 `yaml:"operationId,omitempty"`
	Summary     string                 `yaml:"summary,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Tags        []string               `yaml:"tags,omitempty"`
	Parameters  []Parameter            `yaml:"parameters,omitempty"`
	RequestBody *RequestBody           `yaml:"requestBody,omitempty"`
	Responses   map[string]interface{} `yaml:"responses,omitempty"`
}

// PathItem OpenAPI 路径项
type PathItem struct {
	Get       *Operation `yaml:"get,omitempty"`
	Post      *Operation `yaml:"post,omitempty"`
	Put       *Operation `yaml:"put,omitempty"`
	Patch     *Operation `yaml:"patch,omitempty"`
	Delete    *Operation `yaml:"delete,omitempty"`
	Operation *Operation `yaml:"operation,omitempty"` // 通用操作
}

// Server OpenAPI 服务器
type Server struct {
	URL         string            `yaml:"url"`
	Description string            `yaml:"description,omitempty"`
	Variables   map[string]interface{} `yaml:"variables,omitempty"`
}

// OpenAPI OpenAPI 规范结构
type OpenAPI struct {
	OpenAPI    string                 `yaml:"openapi"`
	Info       map[string]interface{} `yaml:"info"`
	Servers    []Server               `yaml:"servers,omitempty"`
	Paths      map[string]PathItem    `yaml:"paths"`
	Components map[string]interface{} `yaml:"components,omitempty"`
	Tags       []map[string]interface{} `yaml:"tags,omitempty"`
}

// APIEndpoint 解析后的 API 端点
type APIEndpoint struct {
	Path        string
	Method      string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   map[string]interface{}
}

// Parser OpenAPI 解析器
type Parser struct {
	spec *OpenAPI
}

// NewParser 创建新的 OpenAPI 解析器
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile 从文件解析 OpenAPI 规范
func (p *Parser) ParseFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read OpenAPI file: %w", err)
	}

	return p.ParseYAML(data)
}

// ParseURL 从 URL 解析 OpenAPI 规范（待实现 HTTP 客户端）
func (p *Parser) ParseURL(url string) error {
	// TODO: 实现 HTTP 获取
	return fmt.Errorf("ParseURL not yet implemented")
}

// ParseYAML 从 YAML 字节解析 OpenAPI 规范
func (p *Parser) ParseYAML(data []byte) error {
	var spec OpenAPI
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("failed to parse OpenAPI YAML: %w", err)
	}

	p.spec = &spec
	return nil
}

// GetEndpoints 获取所有 API 端点
func (p *Parser) GetEndpoints() []APIEndpoint {
	if p.spec == nil {
		return nil
	}

	var endpoints []APIEndpoint

	for path, pathItem := range p.spec.Paths {
		// 为每个 HTTP 方法创建端点
		methods := map[string]*Operation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"PUT":    pathItem.Put,
			"PATCH":  pathItem.Patch,
			"DELETE": pathItem.Delete,
		}

		for method, op := range methods {
			if op != nil {
				endpoint := APIEndpoint{
					Path:        path,
					Method:      method,
					OperationID: op.OperationID,
					Summary:     op.Summary,
					Description: op.Description,
					Tags:        op.Tags,
					Parameters:  op.Parameters,
					RequestBody: op.RequestBody,
					Responses:   op.Responses,
				}

				// 如果没有 operationId，生成一个
				if endpoint.OperationID == "" {
					endpoint.OperationID = p.generateOperationID(path, method)
				}

				endpoints = append(endpoints, endpoint)
			}
		}
	}

	return endpoints
}

// GetBaseURL 获取基础 URL
func (p *Parser) GetBaseURL() string {
	if p.spec == nil || len(p.spec.Servers) == 0 {
		return ""
	}
	return p.spec.Servers[0].URL
}

// GetInfo 获取 API 信息
func (p *Parser) GetInfo() map[string]interface{} {
	if p.spec == nil {
		return nil
	}
	return p.spec.Info
}

// generateOperationID 生成 operationId
func (p *Parser) generateOperationID(path, method string) string {
	// 清理路径参数
	cleanPath := strings.ReplaceAll(path, "{", "")
	cleanPath = strings.ReplaceAll(cleanPath, "}", "")

	// 分割路径
	parts := strings.Split(strings.Trim(cleanPath, "/"), "/")

	// 转换为驼峰命名
	var result strings.Builder
	result.WriteString(strings.ToLower(method))

	for _, part := range parts {
		if part != "" {
			// 将路径部分转换为驼峰
			runes := []rune(part)
			if len(runes) > 0 {
				runes[0] = rune(strings.ToUpper(string(runes[0]))[0])
				result.WriteString(string(runes))
			}
		}
	}

	return result.String()
}

// GetSpec 获取原始 OpenAPI 规范
func (p *Parser) GetSpec() *OpenAPI {
	return p.spec
}
