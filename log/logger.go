package log

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志记录器结构体
type Logger struct {
	logger        *zap.Logger
	sugar         *zap.SugaredLogger
	config        Config
	flushTicker   *time.Ticker // 定时刷新器
	stopFlushChan chan bool    // 停止刷新通道
}

// DefaultConfig 返回默认的日志配置
func DefaultConfig() Config {
	return Config{
		Level:         INFO,   // 默认INFO级别
		StdoutEnable:  true,   // 默认输出到标准输出
		FileOutEnable: false,  // 默认不输出到文件
		OutputDir:     "",     // 默认输出到标准输出
		Filename:      "",     // 默认日志文件名，如app.log
		ErrorSperate:  false,  // 默认不分离错误日志
		ErrorFilename: "",     // 默认错误日志文件名，如error.log
		MaxSize:       100,    // 默认100MB
		MaxAge:        7,      // 默认保留7天
		ByDate:        false,  // 默认按日期分文件
		Development:   false,  // 默认生产模式
		Encoding:      "json", // 默认JSON格式
		BufferSize:    0,      // 默认使用zap默认缓冲区大小
		FlushInterval: 2,      // 默认5秒自动刷新一次
		FlushOnWrite:  false,  // 默认不在每次写入后立即刷新
	}
}

// CustomLevelEnabler 自定义日志级别过滤器
// 只允许指定范围内的日志级别通过
// minLevel: 允许的最小级别
// maxLevel: 允许的最大级别
type CustomLevelEnabler struct {
	minLevel zapcore.Level
	maxLevel zapcore.Level
}

// Enabled 实现LevelEnabler接口，判断日志级别是否允许通过
func (e CustomLevelEnabler) Enabled(level zapcore.Level) bool {
	return level >= e.minLevel && level <= e.maxLevel
}

// New 创建一个新的日志记录器
func New(config Config) (*Logger, error) {
	// 如果配置未指定，则使用默认配置
	if config.Filename == "" {
		config.Filename = "app.log"
	}
	if config.ErrorFilename == "" {
		config.ErrorFilename = "error_" + config.Filename
	}
	if config.MaxSize <= 0 {
		config.MaxSize = 100
	}
	if config.MaxAge <= 0 {
		config.MaxAge = 7
	}
	if config.Encoding == "" {
		config.Encoding = "json"
	}

	// 设置日志级别
	atomicLevel := zap.NewAtomicLevelAt(config.Level.ToZapLevel())

	// 创建编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if config.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 准备core列表
	var cores []zapcore.Core

	// 标准输出core
	if config.StdoutEnable {
		stdoutCore := zapcore.NewCore(
			encoder,
			zapcore.Lock(os.Stdout),
			atomicLevel,
		)
		cores = append(cores, stdoutCore)
	}

	// 文件输出core
	if config.FileOutEnable {
		// 确保输出目录存在
		err := os.MkdirAll(config.OutputDir, 0755)
		if err != nil {
			return nil, err
		}

		// 生成正常日志文件名
		normalFilename := generateFilename(config)
		normalFilePath := config.OutputDir + "/" + normalFilename

		// 创建正常日志文件writer
		normalWriter, err := newLogWriter(normalFilePath, config.MaxSize, config.MaxAge)
		if err != nil {
			return nil, err
		}

		if config.ErrorSperate {
			// 如果开启错误日志分离
			// 生成错误日志文件名
			errorFilename := generateFilenameError(config)
			errorFilePath := config.OutputDir + "/" + errorFilename

			// 创建错误日志文件writer
			errorWriter, err := newLogWriter(errorFilePath, config.MaxSize, config.MaxAge)
			if err != nil {
				return nil, err
			}

			// 创建正常日志core：只记录Debug、Info、Warn
			normalEnabler := CustomLevelEnabler{
				minLevel: zapcore.DebugLevel,
				maxLevel: zapcore.WarnLevel,
			}
			normalCore := zapcore.NewCore(
				encoder,
				zapcore.Lock(normalWriter),
				normalEnabler,
			)

			// 创建错误日志core：只记录Error、Fatal
			errorEnabler := CustomLevelEnabler{
				minLevel: zapcore.ErrorLevel,
				maxLevel: zapcore.FatalLevel,
			}
			errorCore := zapcore.NewCore(
				encoder,
				zapcore.Lock(errorWriter),
				errorEnabler,
			)

			// 添加正常日志core和错误日志core
			cores = append(cores, normalCore, errorCore)
		} else {
			// 如果不开启错误日志分离，所有日志都输出到正常日志文件
			allCore := zapcore.NewCore(
				encoder,
				zapcore.Lock(normalWriter),
				atomicLevel,
			)
			cores = append(cores, allCore)
		}
	}

	// 如果没有配置任何core，添加默认的stdout core
	if len(cores) == 0 {
		defaultCore := zapcore.NewCore(
			encoder,
			zapcore.Lock(os.Stdout),
			atomicLevel,
		)
		cores = append(cores, defaultCore)
	}

	// 组合core
	core := zapcore.NewTee(cores...)

	// 添加caller和stacktrace
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	// 创建logger
	zapLogger := zap.New(core, options...)

	logger := &Logger{
		logger:        zapLogger,
		sugar:         zapLogger.Sugar(),
		config:        config,
		stopFlushChan: make(chan bool),
	}

	// 如果配置了自动刷新间隔，启动定时刷新
	if config.FlushInterval > 0 {
		logger.startAutoFlush()
	}

	return logger, nil
}

// generateFilenameError 根据配置生成错误日志文件名
func generateFilenameError(config Config) string {
	if config.ByDate {
		// 如果启用按日期分文件，添加日期后缀
		dateStr := time.Now().Format("20060102")
		// 检查文件名是否已有扩展名
		baseName := config.ErrorFilename
		extension := ""
		if dotIndex := len(baseName) - 4; dotIndex > 0 && baseName[dotIndex:] == ".log" {
			baseName = baseName[:dotIndex]
			extension = ".log"
		}
		return fmt.Sprintf("%s.%s%s", baseName, dateStr, extension)
	}
	return config.ErrorFilename
}

// newLogWriter 创建一个日志文件writer
func newLogWriter(filePath string, maxSize, maxAge int) (zapcore.WriteSyncer, error) {
	// 这里可以添加日志文件轮转逻辑
	// 目前简单实现，直接返回文件writer
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return zapcore.AddSync(file), nil
}

// SetLevel 设置日志记录的最低级别
func (l *Logger) SetLevel(level Level) {
	l.logger.Core().Enabled(level.ToZapLevel())
}

// GetLevel 获取当前日志记录的最低级别
func (l *Logger) GetLevel() Level {
	// zap没有直接获取当前级别的方法，这里返回配置中的级别
	return l.config.Level
}

// startAutoFlush 启动自动刷新
func (l *Logger) startAutoFlush() {
	l.flushTicker = time.NewTicker(time.Duration(l.config.FlushInterval) * time.Second)

	go func() {
		for {
			select {
			case <-l.flushTicker.C:
				// 定期刷新日志缓冲区
				l.logger.Sync()
			case <-l.stopFlushChan:
				// 停止自动刷新
				l.flushTicker.Stop()
				return
			}
		}
	}()
}

// Close 关闭日志记录器，释放资源
func (l *Logger) Close() error {
	// 停止自动刷新
	if l.flushTicker != nil {
		l.stopFlushChan <- true
	}

	// 确保所有日志都写入磁盘
	return l.logger.Sync()
}

// Debugw 记录调试级别结构化日志
func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Debugwc 记录带context的调试级别结构化日志，自动添加traceid和spanid
func (l *Logger) Debugwc(ctx context.Context, msg string, keysAndValues ...interface{}) {
	// 检测context中的span，添加traceid和spanid
	keysAndValues = l.addTraceContext(ctx, keysAndValues...)
	l.sugar.Debugw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Infow 记录信息级别结构化日志
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Infowc 记录带context的信息级别结构化日志，自动添加traceid和spanid
func (l *Logger) Infowc(ctx context.Context, msg string, keysAndValues ...interface{}) {
	// 检测context中的span，添加traceid和spanid
	keysAndValues = l.addTraceContext(ctx, keysAndValues...)
	l.sugar.Infow(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Warningw 记录警告级别结构化日志
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Warnwc 记录带context的警告级别结构化日志，自动添加traceid和spanid
func (l *Logger) Warnwc(ctx context.Context, msg string, keysAndValues ...interface{}) {
	// 检测context中的span，添加traceid和spanid
	keysAndValues = l.addTraceContext(ctx, keysAndValues...)
	l.sugar.Warnw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Errorw 记录错误级别结构化日志
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Errorwc 记录带context的错误级别结构化日志，自动添加traceid和spanid
func (l *Logger) Errorwc(ctx context.Context, msg string, keysAndValues ...interface{}) {
	// 检测context中的span，添加traceid和spanid
	keysAndValues = l.addTraceContext(ctx, keysAndValues...)
	l.sugar.Errorw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Fatalw 记录致命级别结构化日志，并终止程序
func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.sugar.Fatalw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Fatalwc 记录带context的致命级别结构化日志，自动添加traceid和spanid
func (l *Logger) Fatalwc(ctx context.Context, msg string, keysAndValues ...interface{}) {
	// 检测context中的span，添加traceid和spanid
	keysAndValues = l.addTraceContext(ctx, keysAndValues...)
	l.sugar.Fatalw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Debug 记录调试级别日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Debugfc 记录带context的调试级别日志，自动添加traceid和spanid
func (l *Logger) Debugfc(ctx context.Context, format string, args ...interface{}) {
	// 对于格式化日志，我们只能记录消息，然后单独记录trace信息
	// 这里我们创建一个新的sugar logger，添加traceid和spanid
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		// 使用With方法创建带有trace信息的logger
		l.withTrace(traceID, spanID).Debugf(format, args...)
	} else {
		l.sugar.Debugf(format, args...)
	}
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Info 记录信息级别日志
func (l *Logger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Infofc 记录带context的信息级别日志，自动添加traceid和spanid
func (l *Logger) Infofc(ctx context.Context, format string, args ...interface{}) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		l.withTrace(traceID, spanID).Infof(format, args...)
	} else {
		l.sugar.Infof(format, args...)
	}
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Warning 记录警告级别日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Warnfc 记录带context的警告级别日志，自动添加traceid和spanid
func (l *Logger) Warnfc(ctx context.Context, format string, args ...interface{}) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		l.withTrace(traceID, spanID).Warnf(format, args...)
	} else {
		l.sugar.Warnf(format, args...)
	}
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Error 记录错误级别日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Errorfc 记录带context的错误级别日志，自动添加traceid和spanid
func (l *Logger) Errorfc(ctx context.Context, format string, args ...interface{}) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		l.withTrace(traceID, spanID).Errorf(format, args...)
	} else {
		l.sugar.Errorf(format, args...)
	}
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Fatal 记录致命级别日志，并终止程序
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Fatalfc 记录带context的致命级别日志，自动添加traceid和spanid
func (l *Logger) Fatalfc(ctx context.Context, format string, args ...interface{}) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		l.withTrace(traceID, spanID).Fatalf(format, args...)
	} else {
		l.sugar.Fatalf(format, args...)
	}
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// With 添加字段到日志记录器，返回新的日志记录器
func (l *Logger) With(keysAndValues ...interface{}) *Logger {
	return &Logger{
		logger: l.sugar.With(keysAndValues...).Desugar(),
		sugar:  l.sugar.With(keysAndValues...),
		config: l.config,
	}
}

// Named 添加名称到日志记录器，返回新的日志记录器
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		logger: l.logger.Named(name),
		sugar:  l.sugar.Named(name),
		config: l.config,
	}
}

// addTraceContext 从context中提取trace和span信息，添加到keysAndValues中
func (l *Logger) addTraceContext(ctx context.Context, keysAndValues ...interface{}) []interface{} {
	// 从context中获取span
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		// 创建新的切片，保留原有容量并添加trace信息
		newKeysAndValues := make([]interface{}, 0, len(keysAndValues)+4) // +4 for traceid and spanid (key-value pairs)
		newKeysAndValues = append(newKeysAndValues, keysAndValues...)
		// 添加traceid和spanid
		newKeysAndValues = append(newKeysAndValues,
			"trace_id", span.SpanContext().TraceID().String(),
			"span_id", span.SpanContext().SpanID().String(),
		)
		return newKeysAndValues
	}
	// 如果没有span，返回原切片
	return keysAndValues
}

// withTrace 创建带有traceid和spanid的sugar logger
func (l *Logger) withTrace(traceID, spanID string) *zap.SugaredLogger {
	return l.sugar.With(
		"trace_id", traceID,
		"span_id", spanID,
	)
}

// generateFilename 根据配置生成日志文件名
func generateFilename(config Config) string {
	if config.ByDate {
		// 如果启用按日期分文件，添加日期后缀
		dateStr := time.Now().Format("20060102")
		// 检查文件名是否已有扩展名
		baseName := config.Filename
		extension := ""
		if dotIndex := len(baseName) - 4; dotIndex > 0 && baseName[dotIndex:] == ".log" {
			baseName = baseName[:dotIndex]
			extension = ".log"
		}
		return fmt.Sprintf("%s.%s%s", baseName, dateStr, extension)
	}
	return config.Filename
}
