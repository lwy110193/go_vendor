package log

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

// Level 表示日志级别
type Level int

// 日志级别常量定义
const (
	// DEBUG 调试级别，最详细的日志信息
	DEBUG Level = iota
	// INFO 信息级别，记录程序运行的基本信息
	INFO
	// WARNING 警告级别，记录可能的问题，但不影响程序继续运行
	WARNING
	// ERROR 错误级别，记录错误信息，程序可能无法正常执行某些功能
	ERROR
	// FATAL 致命级别，记录严重错误，程序可能需要终止
	FATAL
)

// String 返回日志级别的字符串表示
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 根据字符串解析日志级别
func ParseLevel(levelStr string) Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING", "WARN":
		return WARNING
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO // 默认返回INFO级别
	}
}

// ToZapLevel 将自定义Level转换为zap的Level
func (l Level) ToZapLevel() zapcore.Level {
	switch l {
	case DEBUG:
		return zapcore.DebugLevel
	case INFO:
		return zapcore.InfoLevel
	case WARNING:
		return zapcore.WarnLevel
	case ERROR:
		return zapcore.ErrorLevel
	case FATAL:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}