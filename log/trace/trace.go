package trace

import (
	"context"
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

	traceID      string
	spanID       string
	parentSpanID string
	startTime    time.Time
	endTime      time.Time
	attributes   map[string]any
}

// ContextWithTrace 将链路写入上下文。
func ContextWithTrace(ctx context.Context, trace *Trace) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if trace == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxValueKey, trace)
}

// NewTrace 根据上下文创建日志链路。
func NewTrace(ctx context.Context, attributes ...map[string]any) *Trace {
	if parent := traceFromContext(ctx); parent != nil {
		return parent.NewChild(attributes...)
	}
	return newTrace("", attributes...)
}

// NewChild 创建当前链路下的子链路。
func (t *Trace) NewChild(attributes ...map[string]any) *Trace {
	if t == nil {
		return NewTrace(context.Background(), attributes...)
	}

	t.mu.RLock()
	traceID := t.traceID
	parentSpanID := t.spanID
	t.mu.RUnlock()

	child := newTrace(parentSpanID, attributes...)
	child.traceID = traceID
	return child
}

func traceFromContext(ctx context.Context) *Trace {
	if ctx == nil {
		return nil
	}

	trace, _ := ctx.Value(ctxValueKey).(*Trace)
	return trace
}

func newTrace(parentSpanID string, attributes ...map[string]any) *Trace {
	t := &Trace{
		traceID:      uuid.NewString(),
		spanID:       uuid.NewString(),
		parentSpanID: parentSpanID,
		startTime:    time.Now(),
		attributes:   make(map[string]any),
	}

	for _, attrs := range attributes {
		for key, value := range attrs {
			t.attributes[key] = value
		}
	}

	return t
}

// End 结束当前链路并记录结束时间。
func (t *Trace) End() {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.endTime.IsZero() {
		t.endTime = time.Now()
	}
}

// TraceID 返回链路 ID。
func (t *Trace) TraceID() string {
	if t == nil {
		return ""
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.traceID
}

// SpanID 返回当前节点 ID。
func (t *Trace) SpanID() string {
	if t == nil {
		return ""
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.spanID
}

// ParentSpanID 返回父节点 ID。
func (t *Trace) ParentSpanID() string {
	if t == nil {
		return ""
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.parentSpanID
}

// StartTime 返回链路开始时间。
func (t *Trace) StartTime() time.Time {
	if t == nil {
		return time.Time{}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.startTime
}

// EndTime 返回链路结束时间。
func (t *Trace) EndTime() time.Time {
	if t == nil {
		return time.Time{}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.endTime
}

// Duration 返回链路耗时；未结束时返回从开始到当前的耗时。
func (t *Trace) Duration() time.Duration {
	if t == nil {
		return 0
	}

	t.mu.RLock()
	startTime := t.startTime
	endTime := t.endTime
	t.mu.RUnlock()

	if startTime.IsZero() {
		return 0
	}
	if endTime.IsZero() {
		return time.Since(startTime)
	}
	return endTime.Sub(startTime)
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
	fields["span_id"] = t.spanID
	fields["parent_span_id"] = t.parentSpanID
	fields["start_time"] = t.startTime
	fields["end_time"] = t.endTime
	fields["duration"] = t.durationLocked()
	for key, value := range t.attributes {
		fields[key] = value
	}
	return fields
}

func (t *Trace) durationLocked() time.Duration {
	if t.startTime.IsZero() {
		return 0
	}
	if t.endTime.IsZero() {
		return time.Since(t.startTime)
	}
	return t.endTime.Sub(t.startTime)
}
