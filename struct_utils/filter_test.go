package struct_utils

import (
	"testing"
)

// TestStruct 测试用结构体
type TestStruct struct {
	A string
	B int
	C float64
	D bool
}

func TestFilter(t *testing.T) {
	// 创建测试数据
	testData := []TestStruct{
		{A: "a", B: 1, C: 1.1, D: true},
		{A: "b", B: 5, C: 2.2, D: false},
		{A: "a", B: 3, C: 3.3, D: true},
		{A: "c", B: 6, C: 4.4, D: false},
	}

	// 测试条件1：A等于"a"
	filter1 := []FilterCondition{
		{FieldName: "A", Operator: Equal, Value: "a"},
	}
	result1 := Filter(testData, filter1...)
	if len(result1) != 2 {
		t.Errorf("Test 1 failed: expected 2 results, got %d", len(result1))
	}

	// 测试条件2：B大于4
	filter2 := []FilterCondition{
		{FieldName: "B", Operator: GreaterThan, Value: 4},
	}
	result2 := Filter(testData, filter2...)
	if len(result2) != 2 {
		t.Errorf("Test 2 failed: expected 2 results, got %d", len(result2))
	}

	// 测试条件3：A等于"a"且B大于2
	filter3 := []FilterCondition{
		{FieldName: "A", Operator: Equal, Value: "a"},
		{FieldName: "B", Operator: GreaterThan, Value: 2},
	}
	result3 := Filter(testData, filter3...)
	if len(result3) != 1 {
		t.Errorf("Test 3 failed: expected 1 result, got %d", len(result3))
	}

	// 测试条件4：C小于等于3.3
	filter4 := []FilterCondition{
		{FieldName: "C", Operator: LessThanOrEqual, Value: 3.3},
	}
	result4 := Filter(testData, filter4...)
	if len(result4) != 3 {
		t.Errorf("Test 4 failed: expected 3 results, got %d", len(result4))
	}

	// 测试条件5：D等于true
	filter5 := []FilterCondition{
		{FieldName: "D", Operator: Equal, Value: true},
	}
	result5 := Filter(testData, filter5...)
	if len(result5) != 2 {
		t.Errorf("Test 5 failed: expected 2 results, got %d", len(result5))
	}
}
