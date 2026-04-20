package registry

import (
	"fmt"
	"sync"

	"github.com/sundy-yao/mcp-for-swagger/internal/mcp/types"
)

// ToolRegistry 工具注册表
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*types.ToolDefinition
}

// NewToolRegistry 创建新的工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*types.ToolDefinition),
	}
}

// globalRegistry 全局工具注册表实例
var (
	globalRegistry *ToolRegistry
	registryOnce   sync.Once
)

// GetGlobalRegistry 获取全局工具注册表
func GetGlobalRegistry() *ToolRegistry {
	registryOnce.Do(func() {
		globalRegistry = NewToolRegistry()
	})
	return globalRegistry
}

// SetGlobalRegistry 设置全局工具注册表（用于测试或重置）
func SetGlobalRegistry(r *ToolRegistry) {
	globalRegistry = r
}

// Register 注册一个工具
func (r *ToolRegistry) Register(name, description string, inputSchema map[string]interface{}, handler types.ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[name] = &types.ToolDefinition{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		Handler:     handler,
	}
}

// Unregister 注销一个工具
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// GetTool 获取工具定义
func (r *ToolRegistry) GetTool(name string) (*types.ToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// GetTools 获取所有工具（用于 tools/list）
func (r *ToolRegistry) GetTools() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	return result
}

// ListTools 列出所有工具名称
func (r *ToolRegistry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// CallTool 调用工具
func (r *ToolRegistry) CallTool(name string, args map[string]interface{}) (interface{}, error) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool.Handler(args)
}

// Clear 清空所有工具
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = make(map[string]*types.ToolDefinition)
}

// Count 获取工具数量
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}
