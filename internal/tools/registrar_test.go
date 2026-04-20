package tools

import (
	"testing"
)

func TestConvertValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		schema   interface{}
		expected interface{}
	}{
		{
			name:     "string to integer",
			value:    "1",
			schema:   map[string]interface{}{"type": "integer"},
			expected: 1,
		},
		{
			name:     "string to integer with priority value",
			value:    "1",
			schema:   map[string]interface{}{"type": "integer", "enum": []interface{}{1, 2, 3, 4}},
			expected: 1,
		},
		{
			name:     "already integer",
			value:    1,
			schema:   map[string]interface{}{"type": "integer"},
			expected: 1,
		},
		{
			name:     "string to number",
			value:    "3.14",
			schema:   map[string]interface{}{"type": "number"},
			expected: 3.14,
		},
		{
			name:     "string to boolean",
			value:    "true",
			schema:   map[string]interface{}{"type": "boolean"},
			expected: true,
		},
		{
			name:     "string stays string",
			value:    "hello",
			schema:   map[string]interface{}{"type": "string"},
			expected: "hello",
		},
		{
			name:     "nil schema",
			value:    "test",
			schema:   nil,
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertValue(tt.value, tt.schema)
			if result != tt.expected {
				t.Errorf("convertValue(%v, %v) = %v (type: %T), expected %v (type: %T)",
					tt.value, tt.schema, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestBuildFieldSchemaMap(t *testing.T) {
	// 模拟 openapi.yaml 中的 CreateTicketRequest schema
	requestBody := map[string]interface{}{
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"required": []interface{}{"title", "description", "category", "priority"},
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "工单标题",
						},
						"priority": map[string]interface{}{
							"type":        "integer",
							"description": "紧急程度 (1:低 2:中 3:高 4:紧急)",
							"enum":        []interface{}{1, 2, 3, 4},
						},
					},
				},
			},
		},
	}

	// 模拟 schema 提取逻辑
	content := requestBody["content"].(map[string]interface{})
	appJSON := content["application/json"].(map[string]interface{})
	schemaMap := appJSON["schema"].(map[string]interface{})
	props := schemaMap["properties"].(map[string]interface{})

	// 验证 priority 的 schema
	prioritySchema, ok := props["priority"].(map[string]interface{})
	if !ok {
		t.Fatal("priority schema is not a map")
	}

	typ, ok := prioritySchema["type"].(string)
	if !ok {
		t.Fatal("priority type is not a string")
	}

	if typ != "integer" {
		t.Errorf("priority type = %s, expected integer", typ)
	}

	t.Logf("priority schema: %+v", prioritySchema)

	// 测试转换
	result := convertValue("1", prioritySchema)
	if result != 1 {
		t.Errorf("convertValue(\"1\", prioritySchema) = %v (type: %T), expected 1 (type: int)", result, result)
	}
}
