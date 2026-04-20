package openapi

import (
	"testing"
)

func TestParseFile(t *testing.T) {
	parser := NewParser()
	err := parser.ParseFile("../../doc/openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}

	// 验证基本信息
	info := parser.GetInfo()
	if info == nil {
		t.Fatal("Info is nil")
	}

	title, ok := info["title"].(string)
	if !ok || title != "Sample Pet Store API" {
		t.Errorf("Expected title 'Sample Pet Store API', got %v", info["title"])
	}

	// 验证基础 URL
	baseURL := parser.GetBaseURL()
	if baseURL != "http://localhost:8080" {
		t.Errorf("Expected base URL 'http://localhost:8080', got %s", baseURL)
	}

	// 验证端点数量
	endpoints := parser.GetEndpoints()
	expectedCount := 6 // listPets, createPet, getPet, updatePet, deletePet, healthCheck
	if len(endpoints) != expectedCount {
		t.Errorf("Expected %d endpoints, got %d", expectedCount, len(endpoints))
	}
}

func TestGetEndpoints(t *testing.T) {
	parser := NewParser()
	err := parser.ParseFile("../../doc/openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}

	endpoints := parser.GetEndpoints()

	// 验证 listPets 端点
	found := false
	for _, ep := range endpoints {
		if ep.OperationID == "listPets" {
			found = true
			if ep.Method != "GET" {
				t.Errorf("Expected listPets method GET, got %s", ep.Method)
			}
			if ep.Path != "/pets" {
				t.Errorf("Expected listPets path /pets, got %s", ep.Path)
			}
			break
		}
	}
	if !found {
		t.Error("listPets endpoint not found")
	}
}

func TestGenerateOperationID(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		path     string
		method   string
		expected string
	}{
		{"/pets", "GET", "getPets"},
		{"/pets/{petId}", "GET", "getPetsPetId"},
		{"/users/{userId}/posts", "POST", "postUsersUserIdPosts"},
	}

	for _, test := range tests {
		result := parser.generateOperationID(test.path, test.method)
		if result != test.expected {
			t.Errorf("generateOperationID(%s, %s) = %s, expected %s", test.path, test.method, result, test.expected)
		}
	}
}
