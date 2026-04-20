package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sundy-yao/mcp-for-swagger/internal/httpclient"
	"github.com/sundy-yao/mcp-for-swagger/internal/logger"
	"github.com/sundy-yao/mcp-for-swagger/internal/mcp/registry"
	"github.com/sundy-yao/mcp-for-swagger/internal/openapi"
)

// OpenAPIToolRegistrar OpenAPI 工具注册器
type OpenAPIToolRegistrar struct {
	registry     *registry.ToolRegistry
	httpClient   *httpclient.Client
	config       *RegistrarConfig
	fieldSchemas map[string]map[string]interface{} // 保存每个工具的字段 schema 信息
}

// RegistrarConfig 注册器配置
type RegistrarConfig struct {
	ToolPrefix  string
	ExcludeOps  map[string]bool
	IncludeTags map[string]bool
	CustomTools []CustomToolConfig
}

// CustomToolConfig 自定义工具配置
type CustomToolConfig struct {
	Name        string
	Description string
	OperationID string
	Headers     map[string]string
}

// NewOpenAPIToolRegistrar 创建新的 OpenAPI 工具注册器
func NewOpenAPIToolRegistrar(reg *registry.ToolRegistry, client *httpclient.Client, config *RegistrarConfig) *OpenAPIToolRegistrar {
	return &OpenAPIToolRegistrar{
		registry:     reg,
		httpClient:   client,
		config:       config,
		fieldSchemas: make(map[string]map[string]interface{}),
	}
}

// RegisterFromOpenAPI 从 OpenAPI 规范注册工具
func (r *OpenAPIToolRegistrar) RegisterFromOpenAPI(parser *openapi.Parser) error {
	endpoints := parser.GetEndpoints()

	for _, endpoint := range endpoints {
		// 检查是否应该排除
		if r.shouldExclude(endpoint) {
			logger.Info("Skipping excluded endpoint: %s %s", endpoint.Method, endpoint.Path)
			continue
		}

		// 检查 tag 过滤
		if !r.shouldIncludeByTag(endpoint) {
			logger.Info("Skipping endpoint due to tag filter: %s %s", endpoint.Method, endpoint.Path)
			continue
		}

		// 注册工具
		if err := r.registerEndpoint(endpoint); err != nil {
			logger.Error("Failed to register endpoint %s %s: %v", endpoint.Method, endpoint.Path, err)
		}
	}

	logger.Info("Registered %d tools from OpenAPI spec", r.registry.Count())
	return nil
}

// shouldExclude 检查端点是否应该被排除
func (r *OpenAPIToolRegistrar) shouldExclude(endpoint openapi.APIEndpoint) bool {
	if r.config == nil {
		return false
	}

	// 检查 operationId
	if r.config.ExcludeOps[endpoint.OperationID] {
		return true
	}

	// 检查 path
	if r.config.ExcludeOps[endpoint.Path] {
		return true
	}

	return false
}

// shouldIncludeByTag 检查端点是否应该被包含（基于 tag）
func (r *OpenAPIToolRegistrar) shouldIncludeByTag(endpoint openapi.APIEndpoint) bool {
	if r.config == nil || len(r.config.IncludeTags) == 0 {
		return true // 没有配置 tag 过滤时，包含所有
	}

	// 如果配置了 tag 过滤，检查端点的 tags
	if len(endpoint.Tags) == 0 {
		return false // 端点没有 tag，而配置了 tag 过滤，排除
	}

	for _, tag := range endpoint.Tags {
		if r.config.IncludeTags[tag] {
			return true
		}
	}

	return false
}

// registerEndpoint 注册单个端点为 MCP 工具
func (r *OpenAPIToolRegistrar) registerEndpoint(endpoint openapi.APIEndpoint) error {
	// 生成工具名
	toolName := r.buildToolName(endpoint)

	// 生成工具描述
	description := r.buildToolDescription(endpoint)

	// 生成输入 schema
	inputSchema := r.buildInputSchema(endpoint)

	// 保存字段 schema 信息用于类型转换
	r.fieldSchemas[toolName] = r.buildFieldSchemaMap(endpoint)

	// 创建工具处理函数
	handler := r.createHandler(endpoint)

	// 注册工具
	r.registry.Register(toolName, description, inputSchema, handler)

	logger.Info("Registered tool: %s - %s", toolName, endpoint.Summary)
	logger.Debug("[registerEndpoint] fieldSchemas total: %d", len(r.fieldSchemas))
	return nil
}

// buildToolName 构建工具名
func (r *OpenAPIToolRegistrar) buildToolName(endpoint openapi.APIEndpoint) string {
	// 优先使用 operationId
	if endpoint.OperationID != "" {
		if r.config != nil && r.config.ToolPrefix != "" {
			return r.config.ToolPrefix + "_" + endpoint.OperationID
		}
		return endpoint.OperationID
	}

	// 从 path 和 method 生成
	method := strings.ToLower(endpoint.Method)
	path := strings.Trim(endpoint.Path, "/")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	pathParts := strings.Split(path, "/")

	var nameBuilder strings.Builder
	nameBuilder.WriteString(method)
	for _, part := range pathParts {
		if part != "" {
			nameBuilder.WriteString("_")
			nameBuilder.WriteString(part)
		}
	}

	if r.config != nil && r.config.ToolPrefix != "" {
		return r.config.ToolPrefix + "_" + nameBuilder.String()
	}

	return nameBuilder.String()
}

// buildToolDescription 构建工具描述
func (r *OpenAPIToolRegistrar) buildToolDescription(endpoint openapi.APIEndpoint) string {
	var desc strings.Builder

	if endpoint.Summary != "" {
		desc.WriteString(endpoint.Summary)
	}

	if endpoint.Description != "" {
		if desc.Len() > 0 {
			desc.WriteString(" - ")
		}
		desc.WriteString(endpoint.Description)
	}

	if desc.Len() == 0 {
		desc.WriteString(fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path))
	}

	// 添加 tags 信息
	if len(endpoint.Tags) > 0 {
		desc.WriteString(fmt.Sprintf(" (Tags: %s)", strings.Join(endpoint.Tags, ", ")))
	}

	return desc.String()
}

// buildInputSchema 构建输入 JSON Schema
func (r *OpenAPIToolRegistrar) buildInputSchema(endpoint openapi.APIEndpoint) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	// 从路径参数构建 schema
	for _, param := range endpoint.Parameters {
		if param.Required {
			required = append(required, param.Name)
		}

		propSchema := map[string]interface{}{
			"type":        r.getParamType(param.Schema),
			"description": param.Description,
		}
		properties[param.Name] = propSchema
	}

	// 从请求体构建 schema
	if endpoint.RequestBody != nil && endpoint.RequestBody.Content != nil {
		if appJSON, ok := endpoint.RequestBody.Content["application/json"]; ok {
			if schemaMap, ok := appJSON.(map[string]interface{})["schema"]; ok {
				if schema, ok := schemaMap.(map[string]interface{}); ok {
					// 合并 properties
					if props, ok := schema["properties"].(map[string]interface{}); ok {
						for k, v := range props {
							properties[k] = v
						}
					}
					// 合并 required
					if req, ok := schema["required"].([]interface{}); ok {
						for _, r := range req {
							if str, ok := r.(string); ok {
								required = append(required, str)
							}
						}
					}
				}
			}
		}
	}

	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		inputSchema["required"] = required
	}

	return inputSchema
}

// getParamType 从 OpenAPI schema 获取类型字符串
func (r *OpenAPIToolRegistrar) getParamType(schema map[string]interface{}) string {
	if schema == nil {
		return "string"
	}

	if typ, ok := schema["type"].(string); ok {
		return typ
	}

	return "string"
}

// createHandler 创建工具处理函数
func (r *OpenAPIToolRegistrar) createHandler(endpoint openapi.APIEndpoint) func(args map[string]interface{}) (interface{}, error) {
	// 预先计算 toolName 并捕获
	toolName := r.buildToolName(endpoint)
	logger.Debug("[createHandler] toolName=%s, captured endpoint OperationID=%s", toolName, endpoint.OperationID)
	return func(args map[string]interface{}) (interface{}, error) {
		logger.Debug("[handler closure] calling handleEndpoint with toolName=%s, endpoint.OperationID=%s", toolName, endpoint.OperationID)
		return r.handleEndpointWithToolName(endpoint, args, toolName)
	}
}

// buildFieldSchemaMap 构建字段的 schema 映射（用于类型转换）
func (r *OpenAPIToolRegistrar) buildFieldSchemaMap(endpoint openapi.APIEndpoint) map[string]interface{} {
	fieldSchema := make(map[string]interface{})

	// 从路径参数和查询参数收集 schema
	for _, param := range endpoint.Parameters {
		if param.Schema != nil {
			fieldSchema[param.Name] = param.Schema
		}
	}

	// 从请求体收集 schema
	if endpoint.RequestBody != nil && endpoint.RequestBody.Content != nil {
		if appJSON, ok := endpoint.RequestBody.Content["application/json"]; ok {
			if schemaMap, ok := appJSON.(map[string]interface{})["schema"]; ok {
				if schema, ok := schemaMap.(map[string]interface{}); ok {
					if props, ok := schema["properties"].(map[string]interface{}); ok {
						for k, v := range props {
							fieldSchema[k] = v
							logger.Debug("[buildFieldSchemaMap] field %s: schema=%+v", k, v)
						}
					}
				}
			}
		}
	}

	return fieldSchema
}

// convertValue 根据 schema 转换值类型
func convertValue(value interface{}, schema interface{}) interface{} {
	if schema == nil {
		return value
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return value
	}

	typ, ok := schemaMap["type"].(string)
	if !ok {
		return value
	}

	// 如果已经是正确的类型，直接返回
	switch typ {
	case "integer", "int", "number":
		if _, ok := value.(float64); ok {
			return value // JSON 数字已经是 float64
		}
		// 字符串转数字
		if str, ok := value.(string); ok {
			if typ == "integer" || typ == "int" {
				// 尝试解析为整数
				if num, err := strconv.Atoi(str); err == nil {
					return num
				}
				// 如果失败，尝试解析为 float64 然后转 int
				if num, err := strconv.ParseFloat(str, 64); err == nil {
					return int(num)
				}
			} else if typ == "number" {
				if num, err := strconv.ParseFloat(str, 64); err == nil {
					return num
				}
			}
		}
		// 如果 value 是 int，直接返回
		if num, ok := value.(int); ok {
			return num
		}
	case "boolean", "bool":
		if str, ok := value.(string); ok {
			if b, err := strconv.ParseBool(str); err == nil {
				return b
			}
		}
		if b, ok := value.(bool); ok {
			return b
		}
	}

	// 默认保持原值（string 类型不需要转换）
	return value
}

// getFieldSchemaKeys 获取 fieldSchema 的所有键（用于调试）
func getFieldSchemaKeys(schema map[string]interface{}) []string {
	keys := make([]string, 0, len(schema))
	for k := range schema {
		keys = append(keys, k)
	}
	return keys
}

// getFieldSchemasKeys 获取所有已保存的 fieldSchemas 键（用于调试）
func (r *OpenAPIToolRegistrar) getFieldSchemasKeys() []string {
	keys := make([]string, 0, len(r.fieldSchemas))
	for k := range r.fieldSchemas {
		keys = append(keys, k)
	}
	return keys
}

// handleEndpointWithToolName 处理端点调用（带预先计算的 toolName）
func (r *OpenAPIToolRegistrar) handleEndpointWithToolName(endpoint openapi.APIEndpoint, args map[string]interface{}, toolName string) (interface{}, error) {
	ctx := context.Background()

	logger.Debug("[handleEndpointWithToolName] toolName=%s (pre-computed), endpoint.OperationID=%s", toolName, endpoint.OperationID)
	logger.Debug("[handleEndpointWithToolName] fieldSchemas map keys: %v", r.getFieldSchemasKeys())
	logger.Debug("[handleEndpointWithToolName] looking up fieldSchema for toolName: %s", toolName)

	fieldSchema := r.fieldSchemas[toolName]

	logger.Debug("[%s] fieldSchema lookup result: %v (keys: %v)", endpoint.OperationID, fieldSchema, getFieldSchemaKeys(fieldSchema))
	logger.Debug("[%s] input args: %v", endpoint.OperationID, args)

	// 分离路径参数、查询参数和请求体，并进行类型转换
	pathParams := make(map[string]string)
	queryParams := make(map[string]string)
	bodyParams := make(map[string]interface{})

	// 识别哪些参数属于路径，哪些属于查询
	paramNames := make(map[string]bool)
	for _, param := range endpoint.Parameters {
		paramNames[param.Name] = true
		if val, ok := args[param.Name]; ok {
			// 根据 schema 转换类型
			schema := param.Schema
			convertedVal := convertValue(val, schema)
			logger.Debug("[%s] param %s: %v -> %v (type: %T)", endpoint.OperationID, param.Name, val, convertedVal, convertedVal)

			if param.In == "path" {
				pathParams[param.Name] = fmt.Sprintf("%v", convertedVal)
			} else if param.In == "query" {
				queryParams[param.Name] = fmt.Sprintf("%v", convertedVal)
			}
		}
	}

	// 剩余的参数放入请求体，并进行类型转换
	for key, value := range args {
		if !paramNames[key] {
			// 根据 schema 转换类型
			schema := fieldSchema[key]
			convertedVal := convertValue(value, schema)
			logger.Debug("[%s] body param %s: schema=%v, %v -> %v (type: %T)", endpoint.OperationID, key, schema, value, convertedVal, convertedVal)
			bodyParams[key] = convertedVal
		}
	}

	// 替换路径参数
	path := r.replacePathParams(endpoint.Path, pathParams)

	// 构建 HTTP 请求
	req := httpclient.Request{
		Method:      endpoint.Method,
		Path:        path,
		QueryParams: queryParams,
	}

	// 设置请求体（GET 和 DELETE 通常没有请求体）
	if endpoint.Method != http.MethodGet && endpoint.Method != http.MethodDelete {
		if len(bodyParams) > 0 {
			req.Body = bodyParams

			// Debug 日志：输出请求体 JSON
			if bodyJSON, err := json.Marshal(bodyParams); err == nil {
				logger.Debug("[%s] Request body: %s", endpoint.OperationID, string(bodyJSON))
			} else {
				logger.Debug("[%s] Request body marshal error: %v", endpoint.OperationID, err)
			}
		}
	}

	// 执行请求
	resp, err := r.httpClient.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// 解析响应
	var result interface{}
	if err := resp.ParseJSON(&result); err != nil {
		// 如果解析失败，返回原始字符串
		return map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    resp.String(),
			"parse_error": err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"data":        result,
	}, nil
}

// replacePathParams 替换路径中的参数
func (r *OpenAPIToolRegistrar) replacePathParams(path string, params map[string]string) string {
	result := path
	for key, value := range params {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// RegisterCustomTool 注册自定义工具
func (r *OpenAPIToolRegistrar) RegisterCustomTool(config CustomToolConfig, endpoint openapi.APIEndpoint) {
	toolName := config.Name
	handler := func(args map[string]interface{}) (interface{}, error) {
		return r.handleEndpointWithToolName(endpoint, args, toolName)
	}

	inputSchema := r.buildInputSchema(endpoint)

	// 保存字段 schema 信息
	r.fieldSchemas[config.Name] = r.buildFieldSchemaMap(endpoint)

	r.registry.Register(config.Name, config.Description, inputSchema, handler)
	logger.Info("Registered custom tool: %s", config.Name)
}

// GetParamType 从 OpenAPI schema 获取类型字符串（导出函数）
func GetParamType(schema map[string]interface{}) string {
	if schema == nil {
		return "string"
	}

	if typ, ok := schema["type"].(string); ok {
		return typ
	}

	return "string"
}
