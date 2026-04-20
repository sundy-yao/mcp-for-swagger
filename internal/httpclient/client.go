package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client HTTP 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	headers    []string // 多个 HTTP headers，格式："Header-Name:value"
}

// ClientConfig 客户端配置
type ClientConfig struct {
	BaseURL string
	Headers []string // 多个 HTTP headers，格式："Header-Name:value"
	Timeout time.Duration
}

// NewClient 创建新的 HTTP 客户端
func NewClient(config ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		headers: config.Headers,
	}
}

// Request HTTP 请求
type Request struct {
	Method      string
	Path        string
	QueryParams map[string]string
	Headers     map[string]string
	Body        interface{}
}

// Response HTTP 响应
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Do 执行 HTTP 请求
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	// 构建 URL
	reqURL, err := c.buildURL(req.Path, req.QueryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// 构建请求体
	var bodyReader io.Reader
	if req.Body != nil {
		jsonData, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置默认 headers
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("Accept", "application/json")

	// 设置配置的 headers
	for _, header := range c.headers {
		if idx := strings.Index(header, ":"); idx > 0 {
			headerName := strings.TrimSpace(header[:idx])
			headerValue := strings.TrimSpace(header[idx+1:])
			httpReq.Header.Set(headerName, headerValue)
		}
	}

	// 设置自定义 headers（来自请求）
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// 执行请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}, nil
}

// Get 执行 GET 请求
func (c *Client) Get(ctx context.Context, path string, queryParams map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:      http.MethodGet,
		Path:        path,
		QueryParams: queryParams,
	})
}

// Post 执行 POST 请求
func (c *Client) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:      http.MethodPost,
		Path:        path,
		Body:        body,
		Headers:     headers,
	})
}

// Put 执行 PUT 请求
func (c *Client) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:      http.MethodPut,
		Path:        path,
		Body:        body,
		Headers:     headers,
	})
}

// Patch 执行 PATCH 请求
func (c *Client) Patch(ctx context.Context, path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:      http.MethodPatch,
		Path:        path,
		Body:        body,
		Headers:     headers,
	})
}

// Delete 执行 DELETE 请求
func (c *Client) Delete(ctx context.Context, path string, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:      http.MethodDelete,
		Path:        path,
		Headers:     headers,
	})
}

// buildURL 构建完整的 URL
func (c *Client) buildURL(path string, queryParams map[string]string) (string, error) {
	// 确保 path 以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 拼接 baseURL 和 path
	fullURL := c.baseURL + path

	// 添加查询参数
	if len(queryParams) > 0 {
		parsedURL, err := url.Parse(fullURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL: %w", err)
		}

		q := parsedURL.Query()
		for key, value := range queryParams {
			q.Set(key, value)
		}
		parsedURL.RawQuery = q.Encode()
		fullURL = parsedURL.String()
	}

	return fullURL, nil
}

// ParseJSON 解析 JSON 响应
func (r *Response) ParseJSON(v interface{}) error {
	if len(r.Body) == 0 {
		return fmt.Errorf("empty response body")
	}
	return json.Unmarshal(r.Body, v)
}

// String 返回响应体字符串
func (r *Response) String() string {
	return string(r.Body)
}

// IsSuccess 检查响应是否成功
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}
