package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sundy-yao/mcp-for-swagger/internal/config"
	"github.com/sundy-yao/mcp-for-swagger/internal/httpclient"
	"github.com/sundy-yao/mcp-for-swagger/internal/logger"
	"github.com/sundy-yao/mcp-for-swagger/internal/mcp"
	"github.com/sundy-yao/mcp-for-swagger/internal/mcp/registry"
	"github.com/sundy-yao/mcp-for-swagger/internal/openapi"
	"github.com/sundy-yao/mcp-for-swagger/internal/tools"
)

func main() {
	// 确定配置文件路径
	configPath := "config.yaml"
	if envConfig := os.Getenv("CONFIG_PATH"); envConfig != "" {
		configPath = envConfig
	}

	// 加载配置
	settings, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.Init(settings.Log.Level, settings.Log.Dir, settings.Log.File, settings.Log.RetentionDays); err != nil {
		logger.Error("Failed to initialize logger: %v", err)
		os.Exit(1)
	}

	logger.Info("Starting %s v%s on %s:%d", settings.MCP.Name, settings.MCP.Version, settings.MCP.Host, settings.MCP.Port)

	// 解析 OpenAPI 规范
	parser := openapi.NewParser()

	// 优先从文件加载
	if settings.OpenAPI.Path != "" {
		if err := parser.ParseFile(settings.OpenAPI.Path); err != nil {
			logger.Error("Failed to parse OpenAPI file: %v", err)
			os.Exit(1)
		}
		logger.Info("Parsed OpenAPI spec from file: %s", settings.OpenAPI.Path)
	} else if settings.OpenAPI.URL != "" {
		// TODO: 实现从 URL 加载
		logger.Error("Parsing OpenAPI from URL is not yet implemented")
		os.Exit(1)
	} else {
		logger.Error("OpenAPI path or URL not specified in config")
		os.Exit(1)
	}

	// 确定后端 API 基础 URL
	baseURL := settings.OpenAPI.BaseURL
	if baseURL == "" {
		baseURL = parser.GetBaseURL()
	}
	if baseURL == "" {
		logger.Error("Base URL not specified in config and not found in OpenAPI spec")
		os.Exit(1)
	}

	logger.Info("Using backend API base URL: %s", baseURL)

	// 创建 HTTP 客户端
	// 优先使用 headers 配置，如果未配置则兼容使用旧格式 auth_header
	headers := settings.OpenAPI.Headers
	if len(headers) == 0 && settings.OpenAPI.AuthHeader != "" {
		headers = []string{settings.OpenAPI.AuthHeader}
	}
	httpClient := httpclient.NewClient(httpclient.ClientConfig{
		BaseURL: baseURL,
		Headers: headers,
		Timeout: 0, // 使用默认 30 秒
	})

	// 清空全局 registry（确保干净的启动）
	reg := registry.GetGlobalRegistry()
	reg.Clear()

	// 构建注册器配置
	registrarConfig := &tools.RegistrarConfig{
		ToolPrefix:  settings.ToolMap.Prefix,
		ExcludeOps:  make(map[string]bool),
		IncludeTags: make(map[string]bool),
		CustomTools: nil,
	}

	// 设置排除的端点
	for _, exclude := range settings.ToolMap.Exclude {
		registrarConfig.ExcludeOps[exclude] = true
	}

	// 设置包含的 tags
	for _, tag := range settings.ToolMap.IncludeTags {
		registrarConfig.IncludeTags[tag] = true
	}

	// 创建工具注册器
	registrar := tools.NewOpenAPIToolRegistrar(reg, httpClient, registrarConfig)

	// 从 OpenAPI 注册工具
	if err := registrar.RegisterFromOpenAPI(parser); err != nil {
		logger.Error("Failed to register tools from OpenAPI: %v", err)
		os.Exit(1)
	}

	logger.Info("Successfully registered %d tools", reg.Count())

	// 生成服务说明
	instructions := generateInstructions(parser, reg)

	// 创建 MCP 服务器
	server := mcp.NewMCPServer(
		settings.MCP.Name,
		settings.MCP.Version,
		instructions,
		settings.MCP.Host,
		settings.MCP.Port,
	)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在 goroutine 中启动服务器
	go func() {
		if err := server.Run("", 0); err != nil {
			logger.Error("Failed to start server: %v", err)
			os.Exit(1)
		}
	}()

	logger.Info("Server started, waiting for connections...")

	// 等待退出信号
	<-sigChan
	logger.Info("Shutting down server...")

	// 清理
	if err := cleanup(); err != nil {
		logger.Error("Cleanup error: %v", err)
	}
}

// generateInstructions 生成服务说明
func generateInstructions(parser *openapi.Parser, reg *registry.ToolRegistry) string {
	info := parser.GetInfo()

	var title string
	if info != nil {
		if t, ok := info["title"].(string); ok {
			title = t
		}
	}

	tools := reg.GetTools()

	var instructionText string
	if title != "" {
		instructionText = fmt.Sprintf("MCP Service for %s\n\n", title)
	} else {
		instructionText = "MCP OpenAPI Service\n\n"
	}

	instructionText += fmt.Sprintf("Available tools (%d):\n", len(tools))

	for _, tool := range tools {
		name, _ := tool["name"].(string)
		desc, _ := tool["description"].(string)
		instructionText += fmt.Sprintf("- %s: %s\n", name, desc)
	}

	return instructionText
}

// cleanup 清理资源
func cleanup() error {
	// 目前不需要特殊清理
	return nil
}
