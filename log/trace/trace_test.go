package trace

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewTrace(t *testing.T) {
	tr := NewTrace(context.Background(), map[string]any{
		"path": "/api/test",
	})

	assert.NotEmpty(t, tr.TraceID())
	assert.NotEmpty(t, tr.SpanID())
	assert.Empty(t, tr.ParentSpanID())
	assert.NotZero(t, tr.StartTime())
	assert.Zero(t, tr.EndTime())

	_, err := uuid.Parse(tr.TraceID())
	assert.NoError(t, err)
	_, err = uuid.Parse(tr.SpanID())
	assert.NoError(t, err)

	value, ok := tr.Attribute("path")
	assert.True(t, ok)
	assert.Equal(t, "/api/test", value)
}

func TestTraceNewChild(t *testing.T) {
	parent := NewTrace(context.Background())
	child := parent.NewChild(map[string]any{
		"handler": "create",
	})

	assert.Equal(t, parent.TraceID(), child.TraceID())
	assert.Equal(t, parent.SpanID(), child.ParentSpanID())
	assert.NotEqual(t, parent.SpanID(), child.SpanID())

	value, ok := child.Attribute("handler")
	assert.True(t, ok)
	assert.Equal(t, "create", value)
}

func TestNewTraceWithParentFromContext(t *testing.T) {
	parent := NewTrace(context.Background())
	ctx := ContextWithTrace(context.Background(), parent)

	child := NewTrace(ctx, map[string]any{
		"route": "/orders",
	})

	assert.Equal(t, parent.TraceID(), child.TraceID())
	assert.Equal(t, parent.SpanID(), child.ParentSpanID())
	assert.NotEqual(t, parent.SpanID(), child.SpanID())

	value, ok := child.Attribute("route")
	assert.True(t, ok)
	assert.Equal(t, "/orders", value)
}

func TestTraceDuration(t *testing.T) {
	tr := NewTrace(context.Background())

	time.Sleep(time.Millisecond)
	runningDuration := tr.Duration()
	assert.Greater(t, runningDuration, time.Duration(0))

	tr.End()
	endTime := tr.EndTime()
	duration := tr.Duration()
	assert.NotZero(t, endTime)
	assert.Greater(t, duration, time.Duration(0))

	time.Sleep(time.Millisecond)
	assert.Equal(t, duration, tr.Duration())
}

func TestTraceAttributes(t *testing.T) {
	tr := NewTrace(context.Background())
	tr.SetAttribute("user_id", 1001)

	value, ok := tr.Attribute("user_id")
	assert.True(t, ok)
	assert.Equal(t, 1001, value)

	attrs := tr.Attributes()
	attrs["user_id"] = 2002

	value, ok = tr.Attribute("user_id")
	assert.True(t, ok)
	assert.Equal(t, 1001, value)
}

func TestTraceFields(t *testing.T) {
	tr := NewTrace(context.Background(), map[string]any{
		"method": "GET",
	})
	tr.End()

	fields := tr.Fields()
	assert.Equal(t, tr.TraceID(), fields["trace_id"])
	assert.Equal(t, tr.SpanID(), fields["span_id"])
	assert.Equal(t, tr.ParentSpanID(), fields["parent_span_id"])
	assert.Equal(t, "GET", fields["method"])
	assert.IsType(t, time.Duration(0), fields["duration"])
}
