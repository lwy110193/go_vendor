package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/lwy110193/go_vendor/log"
	"github.com/lwy110193/go_vendor/log/trace"
)

// Logger 日志记录器结构体
type Logger struct {
	logger        *zap.Logger
	sugar         *zap.SugaredLogger
	config        log.Config
	flushTicker   *time.Ticker // 定时刷新器
	stopFlushChan chan bool    // 停止刷新通道
	closers       []io.Closer
}

var nowFunc = time.Now

type stdoutWriteSyncer struct {
	*os.File
}

func (stdoutWriteSyncer) Sync() error {
	return nil
}

type dateLogWriter struct {
	mu        sync.Mutex
	outputDir string
	filename  string
	filePath  string
	file      *os.File
}

func newDateLogWriter(outputDir, filename string) (*dateLogWriter, error) {
	w := &dateLogWriter{
		outputDir: outputDir,
		filename:  filename,
	}
	if err := w.rotateLocked(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dateLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateLocked(); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *dateLogWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *dateLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *dateLogWriter) rotateLocked() error {
	nextPath := filepath.Join(w.outputDir, filenameByDate(w.filename, nowFunc()))
	if w.file != nil && w.filePath == nextPath {
		return nil
	}

	nextFile, err := os.OpenFile(nextPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	w.file = nextFile
	w.filePath = nextPath
	return nil
}

// DefaultConfig 返回默认的日志配置
func DefaultConfig() log.Config {
	return log.Config{
		Level:         log.INFO, // 默认INFO级别
		StdoutEnable:  true,     // 默认输出到标准输出
		FileOutEnable: false,    // 默认不输出到文件
		OutputDir:     "",       // 默认输出到标准输出
		Filename:      "",       // 默认日志文件名，如app.log
		ErrorSperate:  false,    // 默认不分离错误日志
		ErrorFilename: "",       // 默认错误日志文件名，如error.log
		MaxSize:       100,      // 默认100MB
		MaxAge:        7,        // 默认保留7天
		ByDate:        false,    // 默认按日期分文件
		Development:   false,    // 默认生产模式
		Encoding:      "json",   // 默认JSON格式
		BufferSize:    0,        // 默认使用zap默认缓冲区大小
		FlushInterval: 2,        // 默认5秒自动刷新一次
		FlushOnWrite:  false,    // 默认不在每次写入后立即刷新
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
func New(config log.Config) (*Logger, error) {
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
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		FunctionKey:   zapcore.OmitKey,
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime:    zapcore.TimeEncoderOfLayout(time.DateTime), //zapcore.RFC3339TimeEncoder,
		// EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if config.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 准备core列表
	var cores []zapcore.Core
	var closers []io.Closer

	// 标准输出core
	if config.StdoutEnable {
		stdoutCore := zapcore.NewCore(
			encoder,
			zapcore.Lock(stdoutWriteSyncer{File: os.Stdout}),
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

		// 创建正常日志文件writer
		normalWriter, err := newConfiguredLogWriter(config, config.Filename)
		if err != nil {
			return nil, err
		}
		closers = appendLogCloser(closers, normalWriter)

		if config.ErrorSperate {
			// 如果开启错误日志分离
			// 创建错误日志文件writer
			errorWriter, err := newConfiguredLogWriter(config, config.ErrorFilename)
			if err != nil {
				return nil, err
			}
			closers = appendLogCloser(closers, errorWriter)

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
			zapcore.Lock(stdoutWriteSyncer{File: os.Stdout}),
			atomicLevel,
		)
		cores = append(cores, defaultCore)
	}

	// 组合core
	core := zapcore.NewTee(cores...)

	// 添加caller和stacktrace
	options := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	// 创建logger
	zapLogger := zap.New(core, options...)

	logger := &Logger{
		logger:        zapLogger,
		sugar:         zapLogger.Sugar(),
		config:        config,
		stopFlushChan: make(chan bool),
		closers:       closers,
	}

	// 如果配置了自动刷新间隔，启动定时刷新
	if config.FlushInterval > 0 {
		logger.startAutoFlush()
	}

	return logger, nil
}

// generateFilenameError 根据配置生成错误日志文件名
func generateFilenameError(config log.Config) string {
	if config.ByDate {
		return filenameByDate(config.ErrorFilename, nowFunc())
	}
	return config.ErrorFilename
}

func newConfiguredLogWriter(config log.Config, filename string) (zapcore.WriteSyncer, error) {
	if config.ByDate {
		return newDateLogWriter(config.OutputDir, filename)
	}
	return newLogWriter(filepath.Join(config.OutputDir, filename), config.MaxSize, config.MaxAge)
}

// newLogWriter 创建一个日志文件writer
func newLogWriter(filePath string, maxSize, maxAge int) (zapcore.WriteSyncer, error) {
	// 这里可以添加日志文件轮转逻辑
	// 目前简单实现，直接返回文件writer
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func appendLogCloser(closers []io.Closer, writer zapcore.WriteSyncer) []io.Closer {
	if closer, ok := writer.(io.Closer); ok {
		return append(closers, closer)
	}
	return closers
}

// SetLevel 设置日志记录的最低级别
func (l *Logger) SetLevel(level log.Level) {
	l.logger.Core().Enabled(level.ToZapLevel())
}

// GetLevel 获取当前日志记录的最低级别
func (l *Logger) GetLevel() log.Level {
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
	err := l.logger.Sync()
	for _, closer := range l.closers {
		if closeErr := closer.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

// Debug 记录调试级别结构化日志
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// DebugCtx 记录带context的调试级别结构化日志，自动添加traceid
func (l *Logger) DebugCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	keysAndValues = l.addTraceInfoFromContext(ctx, keysAndValues...)
	l.sugar.Debugw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Info 记录信息级别结构化日志
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// InfoCtx 记录带context的信息级别结构化日志，自动添加traceid
func (l *Logger) InfoCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	keysAndValues = l.addTraceInfoFromContext(ctx, keysAndValues...)
	l.sugar.Infow(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Warn 记录警告级别结构化日志
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// WarnCtx 记录带context的警告级别结构化日志，自动添加traceid
func (l *Logger) WarnCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	keysAndValues = l.addTraceInfoFromContext(ctx, keysAndValues...)
	l.sugar.Warnw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Error 记录错误级别结构化日志
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// ErrorCtx 记录带context的错误级别结构化日志，自动添加traceid
func (l *Logger) ErrorCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	keysAndValues = l.addTraceInfoFromContext(ctx, keysAndValues...)
	l.sugar.Errorw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// Fatal 记录致命级别结构化日志，并终止程序
func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.sugar.Fatalw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
	os.Exit(1)
}

// FatalCtx 记录带context的致命级别结构化日志，自动添加traceid
func (l *Logger) FatalCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	keysAndValues = l.addTraceInfoFromContext(ctx, keysAndValues...)
	l.sugar.Fatalw(msg, keysAndValues...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
	os.Exit(1)
}

// DebugFormat 记录调试级别日志
func (l *Logger) DebugFormat(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// DebugFormatCtx 记录带context的调试级别日志，自动添加traceid
func (l *Logger) DebugFormatCtx(ctx context.Context, format string, args ...interface{}) {
	l.addTraceInfoFromTrace(ctx).Debugf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// InfoFormat 记录信息级别日志
func (l *Logger) InfoFormat(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// InfoFormatCtx 记录带context的信息级别日志，自动添加traceid
func (l *Logger) InfoFormatCtx(ctx context.Context, format string, args ...interface{}) {
	l.addTraceInfoFromTrace(ctx).Infof(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// WarningFormat 记录警告级别日志
func (l *Logger) WarnFormat(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// WarnFormatCtx 记录带context的警告级别日志，自动添加traceid
func (l *Logger) WarnFormatCtx(ctx context.Context, format string, args ...interface{}) {
	l.addTraceInfoFromTrace(ctx).Warnf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// ErrorFormat 记录错误级别日志
func (l *Logger) ErrorFormat(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// ErrorFormatCtx 记录带context的错误级别日志，自动添加traceid
func (l *Logger) ErrorFormatCtx(ctx context.Context, format string, args ...interface{}) {
	l.addTraceInfoFromTrace(ctx).Errorf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
}

// FatalFormat 记录致命级别日志，并终止程序
func (l *Logger) FatalFormat(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
	os.Exit(1)
}

// FatalFormatCtx 记录带context的致命级别日志，自动添加traceid
func (l *Logger) FatalFormatCtx(ctx context.Context, format string, args ...interface{}) {
	l.addTraceInfoFromTrace(ctx).Fatalf(format, args...)
	if l.config.FlushOnWrite {
		l.logger.Sync()
	}
	os.Exit(1)
}

// addTraceInfoFromContext 从context中提取trace信息，添加到keysAndValues中
func (l *Logger) addTraceInfoFromContext(ctx context.Context, keysAndValues ...interface{}) []interface{} {
	traceID := trace.GetTraceId(ctx)
	parentTraceID := trace.GetParentTraceId(ctx)
	costTime := trace.GetCostTime(ctx)

	newKeysAndValues := make([]interface{}, 0, len(keysAndValues)+6)
	// 添加traceid和spanid
	newKeysAndValues = append(newKeysAndValues,
		"trace_id", traceID,
		"ptrace_id", parentTraceID,
		"cost_time", costTime,
	)
	newKeysAndValues = append(newKeysAndValues, keysAndValues...)
	return newKeysAndValues
}

// addTraceInfoFromTrace 添加trace信息到日志记录器
func (l *Logger) addTraceInfoFromTrace(ctx context.Context) *zap.SugaredLogger {
	traceID := trace.GetTraceId(ctx)
	parentTraceID := trace.GetParentTraceId(ctx)
	costTime := trace.GetCostTime(ctx)

	return l.sugar.With(
		"trace_id", traceID,
		"ptrace_id", parentTraceID,
		"cost_time", costTime,
	)
}

// generateFilename 根据配置生成日志文件名
func generateFilename(config log.Config) string {
	if config.ByDate {
		return filenameByDate(config.Filename, nowFunc())
	}
	return config.Filename
}

func filenameByDate(filename string, now time.Time) string {
	dateStr := now.Format("20060102")
	baseName := filename
	extension := ""
	if dotIndex := len(baseName) - 4; dotIndex > 0 && baseName[dotIndex:] == ".log" {
		baseName = baseName[:dotIndex]
		extension = ".log"
	}
	return fmt.Sprintf("%s.%s%s", baseName, dateStr, extension)
}
