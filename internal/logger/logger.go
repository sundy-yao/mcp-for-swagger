package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	logger     *log.Logger
	logLevel   LogLevel
	levelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
)

// parseLogLevel 解析日志级别字符串
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO // 默认 INFO
	}
}

// Init 初始化日志系统
func Init(level, dir, file string, retentionDays int) error {
	// 设置日志级别
	logLevel = parseLogLevel(level)

	// 创建日志目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(dir, file)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// 同时输出到文件和 stdout
	mw := io.MultiWriter(os.Stdout, logFile)

	// 创建一个全局 logger，使用 calldepth=2 来跳过包装函数
	logger = log.New(mw, "", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

// logf 是内部日志函数，calldepth 指定跳过多少层调用栈
func logf(level LogLevel, prefix, format string, v ...interface{}) {
	// 检查日志级别
	if level < logLevel {
		return // 低于配置级别，直接返回
	}

	if logger != nil {
		// 使用 Output 而不是 Printf，这样可以正确控制 calldepth
		// calldepth=3: logf -> Info/Error/Warn/Debug -> 实际调用者
		msg := fmt.Sprintf(prefix+format, v...)
		logger.Output(3, msg)
	} else {
		log.Printf(prefix+format, v...)
	}
}

// Info 输出 INFO 级别日志
func Info(format string, v ...interface{}) {
	logf(INFO, "[INFO] ", format, v...)
}

// Warn 输出 WARN 级别日志
func Warn(format string, v ...interface{}) {
	logf(WARN, "[WARN] ", format, v...)
}

// Error 输出 ERROR 级别日志
func Error(format string, v ...interface{}) {
	logf(ERROR, "[ERROR] ", format, v...)
}

// Debug 输出 DEBUG 级别日志
func Debug(format string, v ...interface{}) {
	logf(DEBUG, "[DEBUG] ", format, v...)
}
