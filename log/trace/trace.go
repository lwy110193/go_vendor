package trace

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ctxValueKey = "log_trace"

// SetCtxTraceKey 设置上下文中的链路键名。
func SetCtxTraceKey(key string) {
	ctxValueKey = key
}

// Trace 记录一次日志链路的基础信息。
type Trace struct {
	mu sync.RWMutex

	traceID       string
	parentTraceID string
	startTime     time.Time
	attributes    map[string]any
}

func GetTraceId(ctx context.Context) string {
	if trace := traceFromContext(ctx); trace != nil {
		return trace.TraceID()
	}
	return ""
}

func GetParentTraceId(ctx context.Context) string {
	if trace := traceFromContext(ctx); trace != nil {
		return trace.ParentTraceID()
	}
	return ""
}

func GetCostTime(ctx context.Context) time.Duration {
	if trace := traceFromContext(ctx); trace != nil {
		return trace.Duration()
	}
	return 0
}

// NewTrace 根据上下文创建日志链路。
func NewTrace(ctx context.Context, attributes ...map[string]any) context.Context {
	var parentTraceID string
	if parent := traceFromContext(ctx); parent != nil {
		parentTraceID = parent.TraceID()
	}
	return context.WithValue(ctx, ctxValueKey, newTrace(parentTraceID, attributes...))
}

func traceFromContext(ctx context.Context) *Trace {
	if ctx == nil {
		return nil
	}

	trace, _ := ctx.Value(ctxValueKey).(*Trace)
	return trace
}

func newTrace(parentTraceID string, attributes ...map[string]any) *Trace {
	t := &Trace{
		traceID:       strings.ReplaceAll(uuid.NewString(), "-", "")[:16],
		parentTraceID: parentTraceID,
		startTime:     time.Now(),
		attributes:    make(map[string]any),
	}

	for _, attrs := range attributes {
		for key, value := range attrs {
			t.attributes[key] = value
		}
	}

	return t
}

// TraceID 返回链路 ID。
func (t *Trace) TraceID() string {
	return t.traceID
}

// ParentTraceID 返回父节点 ID。
func (t *Trace) ParentTraceID() string {
	return t.parentTraceID
}

// StartTime 返回链路开始时间。
func (t *Trace) StartTime() time.Time {
	return t.startTime
}

// Duration 返回链路耗时；未结束时返回从开始到当前的耗时。
func (t *Trace) Duration() time.Duration {
	return time.Since(t.startTime)
}

// SetAttribute 设置链路属性。
func (t *Trace) SetAttribute(key string, value any) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.attributes[key] = value
}

// Attribute 获取链路属性。
func (t *Trace) Attribute(key string) (any, bool) {
	if t == nil {
		return nil, false
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	value, ok := t.attributes[key]
	return value, ok
}

// Attributes 返回链路属性副本。
func (t *Trace) Attributes() map[string]any {
	if t == nil {
		return nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	attrs := make(map[string]any, len(t.attributes))
	for key, value := range t.attributes {
		attrs[key] = value
	}
	return attrs
}

// Fields 返回适合写入结构化日志的链路字段。
func (t *Trace) Fields() map[string]any {
	if t == nil {
		return nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	fields := make(map[string]any, len(t.attributes)+6)
	fields["trace_id"] = t.traceID
	fields["p_trace_id"] = t.parentTraceID
	fields["start_time"] = t.startTime
	fields["duration"] = t.Duration()
	for key, value := range t.attributes {
		fields[key] = value
	}
	return fields
}
