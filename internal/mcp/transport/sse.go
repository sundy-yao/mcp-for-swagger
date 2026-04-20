package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sundy-yao/mcp-for-swagger/internal/logger"
	"github.com/google/uuid"
)

// SSETransport SSE 传输层
type SSETransport struct {
	mu              sync.RWMutex
	sseQueues       map[string]chan map[string]interface{}
	currentSessions map[string]chan map[string]interface{}
}

// NewSSETransport 创建新的 SSE 传输
func NewSSETransport() *SSETransport {
	return &SSETransport{
		sseQueues:       make(map[string]chan map[string]interface{}),
		currentSessions: make(map[string]chan map[string]interface{}),
	}
}

// SSEEndpoint SSE 端点处理函数
func (t *SSETransport) SSEEndpoint(w http.ResponseWriter, r *http.Request) {
	logger.Debug("[SSE] >>> REQUEST START: method=%s path=%s from=%s", r.Method, r.URL.Path, r.RemoteAddr)
	logger.Info("SSE endpoint called from: %s", r.RemoteAddr)

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 获取 Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.Warn("SSE: Streaming not supported")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	logger.Debug("[SSE] Flusher obtained, creating session")

	// 创建会话 ID
	sessionID := uuid.New().String()
	messageQueue := make(chan map[string]interface{}, 256)

	// 注册会话
	t.mu.Lock()
	t.sseQueues[sessionID] = messageQueue
	t.currentSessions[sessionID] = messageQueue
	t.mu.Unlock()

	logger.Info("SSE connection established: %s (total sessions: %d)", sessionID, len(t.currentSessions))
	logger.Debug("[SSE] <<< SESSION CREATED: %s", sessionID)

	// 清理函数
	defer func() {
		t.mu.Lock()
		delete(t.sseQueues, sessionID)
		delete(t.currentSessions, sessionID)
		t.mu.Unlock()
		close(messageQueue)
		logger.Info("SSE connection closed: %s (remaining sessions: %d)", sessionID, len(t.currentSessions))
	}()

	// 获取基础 URL
	host := r.Host
	if host == "" {
		host = "localhost:8000"
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	endpointURL := fmt.Sprintf("%s://%s/messages", scheme, host)

	// 发送 endpoint 事件
	logger.Debug("[SSE] >>> SENDING ENDPOINT EVENT: %s to session %s", endpointURL, sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointURL)
	flusher.Flush()
	logger.Debug("[SSE] <<< ENDPOINT EVENT SENT: %s", sessionID)

	// 获取请求上下文
	ctx := r.Context()

	// 事件循环
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger.Info("SSE event loop started for session: %s", sessionID)
	logger.Debug("[SSE] >>> ENTERING EVENT LOOP: %s", sessionID)

	for {
		select {
		case <-ctx.Done():
			logger.Info("SSE context done for session: %s", sessionID)
			return
		case msg, ok := <-messageQueue:
			if !ok {
				logger.Info("SSE message queue closed for session: %s", sessionID)
				return
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			// 心跳 - SSE 注释格式，以:开头，后面跟一个空格
			logger.Debug("SSE heartbeat for session: %s", sessionID)
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// SendToAllSessions 发送消息到所有会话
func (t *SSETransport) SendToAllSessions(message map[string]interface{}) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, queue := range t.currentSessions {
		select {
		case queue <- message:
		default:
			// 队列已满，跳过
		}
	}
	return nil
}

// SendToSession 发送消息到指定会话
func (t *SSETransport) SendToSession(sessionID string, message map[string]interface{}) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	queue, exists := t.currentSessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	select {
	case queue <- message:
		return nil
	default:
		return fmt.Errorf("queue full for session: %s", sessionID)
	}
}

// GetSessionCount 获取活跃会话数
func (t *SSETransport) GetSessionCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.currentSessions)
}

// CurrentSessions 获取当前会话（只读）
func (t *SSETransport) CurrentSessions() map[string]chan map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	copy := make(map[string]chan map[string]interface{})
	for k, v := range t.currentSessions {
		copy[k] = v
	}
	return copy
}

// GetSessionIDs 获取所有会话 ID
func (t *SSETransport) GetSessionIDs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ids := make([]string, 0, len(t.currentSessions))
	for id := range t.currentSessions {
		ids = append(ids, id)
	}
	return ids
}
