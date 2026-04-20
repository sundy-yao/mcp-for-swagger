package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := ClientConfig{
		BaseURL: "http://localhost:8080",
		Headers: []string{"Authorization: Bearer test-token"},
		Timeout: 10 * time.Second,
	}

	client := NewClient(config)

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL 'http://localhost:8080', got %s", client.baseURL)
	}

	if len(client.headers) != 1 || client.headers[0] != "Authorization: Bearer test-token" {
		t.Errorf("Expected headers ['Authorization: Bearer test-token'], got %v", client.headers)
	}
}

func TestBuildURL(t *testing.T) {
	client := NewClient(ClientConfig{BaseURL: "http://localhost:8080"})

	tests := []struct {
		path        string
		queryParams map[string]string
		expected    string
	}{
		{"/pets", nil, "http://localhost:8080/pets"},
		{"/pets", map[string]string{"limit": "10"}, "http://localhost:8080/pets?limit=10"},
		{"/pets", map[string]string{"limit": "10", "offset": "5"}, "http://localhost:8080/pets?limit=10&offset=5"},
		{"pets", nil, "http://localhost:8080/pets"},
	}

	for _, test := range tests {
		result, err := client.buildURL(test.path, test.queryParams)
		if err != nil {
			t.Errorf("buildURL(%s, %v) error: %v", test.path, test.queryParams, err)
			continue
		}
		if result != test.expected {
			t.Errorf("buildURL(%s, %v) = %s, expected %s", test.path, test.queryParams, result, test.expected)
		}
	}
}

func TestDo(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pets" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": [{"id": "1", "name": "Fluffy"}]}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})

	// 测试 GET 请求
	resp, err := client.Get(context.Background(), "/pets", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expectedBody := `{"data": [{"id": "1", "name": "Fluffy"}]}`
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(resp.Body))
	}
}

func TestIsSuccess(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{300, false},
		{400, false},
		{404, false},
		{500, false},
	}

	for _, test := range tests {
		resp := &Response{StatusCode: test.statusCode}
		result := resp.IsSuccess()
		if result != test.expected {
			t.Errorf("IsSuccess(%d) = %v, expected %v", test.statusCode, result, test.expected)
		}
	}
}
