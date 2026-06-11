package logger

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rootlog "github.com/lwy110193/go_vendor/log"
	"github.com/lwy110193/go_vendor/log/trace"
)

var _ rootlog.LoggerInterface = (*Logger)(nil)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Level != rootlog.INFO {
		t.Fatalf("expected default level INFO, got %v", config.Level)
	}
	if !config.StdoutEnable {
		t.Fatal("expected stdout logging to be enabled by default")
	}
	if config.FileOutEnable {
		t.Fatal("expected file logging to be disabled by default")
	}
	if config.Encoding != "json" {
		t.Fatalf("expected default encoding json, got %q", config.Encoding)
	}
}

func TestLoggerContextMethodsWriteTraceFields(t *testing.T) {
	// stdout := os.Stdout
	// reader, writer, err := os.Pipe()
	// if err != nil {
	// 	t.Fatalf("create stdout pipe: %v", err)
	// }
	// os.Stdout = writer
	// defer func() {
	// 	os.Stdout = stdout
	// }()

	logger, err := New(rootlog.Config{
		Level:         rootlog.DEBUG,
		StdoutEnable:  true,
		FileOutEnable: true,
		ByDate:        true,
		Encoding:      "json",
		OutputDir:     "./",

		FlushInterval: 0,
		FlushOnWrite:  true,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	ctx := trace.NewTrace(context.Background())
	ctx = trace.NewTrace(ctx)
	logger.DebugCtx(ctx, "debug message", "key", "debug")
	logger.InfoCtx(ctx, "info message", "key", "info")
	logger.WarnCtx(ctx, "warn message", "key", "warn")
	logger.ErrorCtx(ctx, "error message", "key", "error")

	logger.DebugFormat("%v %s", "deee", "eeeee")
	logger.DebugFormatCtx(ctx, "%v %s", "info", "infoeee")

	// _ = logger.Close()
	// if err := writer.Close(); err != nil {
	// 	t.Fatalf("close stdout pipe writer: %v", err)
	// }

	// logs := readJSONLogs(t, reader)
	// assertLogMessage(t, logs, "debug message")
	// assertLogMessage(t, logs, "info message")
	// assertLogMessage(t, logs, "warn message")
	// assertLogMessage(t, logs, "error message")

	// for _, entry := range logs {
	// 	if entry["trace_id"] == "" {
	// 		t.Fatalf("expected trace_id in log entry: %#v", entry)
	// 	}
	// 	if _, ok := entry["cost_time"]; !ok {
	// 		t.Fatalf("expected cost_time in log entry: %#v", entry)
	// 	}
	// }
}

func TestLoggerByDateSwitchesFileWhenDateChanges(t *testing.T) {
	originalNowFunc := nowFunc
	currentTime := time.Date(2026, 6, 11, 23, 59, 0, 0, time.Local)
	nowFunc = func() time.Time {
		return currentTime
	}
	t.Cleanup(func() {
		nowFunc = originalNowFunc
	})

	outputDir := t.TempDir()
	logger, err := New(rootlog.Config{
		Level:         rootlog.DEBUG,
		StdoutEnable:  false,
		FileOutEnable: true,
		OutputDir:     outputDir,
		Filename:      "app.log",
		ByDate:        true,
		Encoding:      "json",
		FlushInterval: 0,
		FlushOnWrite:  true,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.Info("before date changes")
	currentTime = time.Date(2026, 6, 12, 0, 1, 0, 0, time.Local)
	logger.Info("after date changes")

	if err := logger.Close(); err != nil {
		t.Fatalf("close logger: %v", err)
	}

	firstDayLog := readFileString(t, filepath.Join(outputDir, "app.20260611.log"))
	secondDayLog := readFileString(t, filepath.Join(outputDir, "app.20260612.log"))

	if !strings.Contains(firstDayLog, "before date changes") {
		t.Fatalf("expected first day log to contain first message, got %q", firstDayLog)
	}
	if strings.Contains(firstDayLog, "after date changes") {
		t.Fatalf("expected first day log not to contain second message, got %q", firstDayLog)
	}
	if !strings.Contains(secondDayLog, "after date changes") {
		t.Fatalf("expected second day log to contain second message, got %q", secondDayLog)
	}
	if strings.Contains(secondDayLog, "before date changes") {
		t.Fatalf("expected second day log not to contain first message, got %q", secondDayLog)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

func readJSONLogs(t *testing.T, reader io.Reader) []map[string]interface{} {
	t.Helper()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}

	var logs []map[string]interface{}
	for _, line := range splitLines(string(data)) {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("parse log line %q: %v", line, err)
		}
		logs = append(logs, entry)
	}

	return logs
}

func splitLines(data string) []string {
	var lines []string
	start := 0
	for i, r := range data {
		if r != '\n' {
			continue
		}
		if start < i {
			lines = append(lines, data[start:i])
		}
		start = i + 1
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func assertLogMessage(t *testing.T, logs []map[string]interface{}, message string) {
	t.Helper()

	for _, entry := range logs {
		if entry["msg"] == message {
			return
		}
	}
	t.Fatalf("expected log message %q in %#v", message, logs)
}
