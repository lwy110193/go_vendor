package struct_utils

import (
	"testing"
)

// SortTestStruct 测试用结构体
type SortTestStruct struct {
	Name     string
	Age      int
	Score    float64
	RankStr  string
	ValueStr string
}

func TestSort(t *testing.T) {
	// 测试数据
	data := []SortTestStruct{
		{Name: "Alice", Age: 30, Score: 85.5, RankStr: "3", ValueStr: "100"},
		{Name: "Bob", Age: 25, Score: 92.0, RankStr: "1", ValueStr: "20"},
		{Name: "Charlie", Age: 35, Score: 78.5, RankStr: "5", ValueStr: "50"},
		{Name: "David", Age: 28, Score: 88.0, RankStr: "2", ValueStr: "150"},
		{Name: "Eve", Age: 32, Score: 82.5, RankStr: "4", ValueStr: "80"},
	}

	// 测试1：按年龄升序排序
	result1 := Sort(data, SortConfig{FieldName: "Age", Direction: Ascending})
	expected1 := []int{25, 28, 30, 32, 35}
	for i, item := range result1 {
		if item.Age != expected1[i] {
			t.Errorf("Test1 failed: expected Age %d, got %d at index %d", expected1[i], item.Age, i)
		}
	}

	// 测试2：按分数降序排序
	result2 := Sort(data, SortConfig{FieldName: "Score", Direction: Descending})
	expected2 := []float64{92.0, 88.0, 85.5, 82.5, 78.5}
	for i, item := range result2 {
		if item.Score != expected2[i] {
			t.Errorf("Test2 failed: expected Score %.1f, got %.1f at index %d", expected2[i], item.Score, i)
		}
	}

	// 测试3：按名字升序排序
	result3 := Sort(data, SortConfig{FieldName: "Name", Direction: Ascending})
	expected3 := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
	for i, item := range result3 {
		if item.Name != expected3[i] {
			t.Errorf("Test3 failed: expected Name %s, got %s at index %d", expected3[i], item.Name, i)
		}
	}

	// 测试4：按RankStr字符串排序（字典序）
	result4 := Sort(data, SortConfig{FieldName: "RankStr", Direction: Ascending})
	expected4 := []string{"1", "2", "3", "4", "5"}
	for i, item := range result4 {
		if item.RankStr != expected4[i] {
			t.Errorf("Test4 failed: expected RankStr %s, got %s at index %d", expected4[i], item.RankStr, i)
		}
	}

	// 测试5：按RankStr转换为数值排序
	result5 := Sort(data, SortConfig{FieldName: "RankStr", Direction: Ascending, ParseStringAsNumber: true})
	expected5 := []string{"1", "2", "3", "4", "5"}
	for i, item := range result5 {
		if item.RankStr != expected5[i] {
			t.Errorf("Test5 failed: expected RankStr %s, got %s at index %d", expected5[i], item.RankStr, i)
		}
	}

	// 测试6：按ValueStr转换为数值降序排序
	result6 := Sort(data, SortConfig{FieldName: "ValueStr", Direction: Descending, ParseStringAsNumber: true})
	expected6 := []string{"150", "100", "80", "50", "20"}
	for i, item := range result6 {
		if item.ValueStr != expected6[i] {
			t.Errorf("Test6 failed: expected ValueStr %s, got %s at index %d", expected6[i], item.ValueStr, i)
		}
	}

	// 测试7：多字段排序（先按年龄升序，再按分数降序）
	result7 := Sort(data,
		SortConfig{FieldName: "Age", Direction: Ascending},
		SortConfig{FieldName: "Score", Direction: Descending})
	expected7 := []SortTestStruct{
		{Name: "Bob", Age: 25, Score: 92.0},
		{Name: "David", Age: 28, Score: 88.0},
		{Name: "Alice", Age: 30, Score: 85.5},
		{Name: "Eve", Age: 32, Score: 82.5},
		{Name: "Charlie", Age: 35, Score: 78.5},
	}
	for i, item := range result7 {
		if item.Name != expected7[i].Name || item.Age != expected7[i].Age || item.Score != expected7[i].Score {
			t.Errorf("Test7 failed: expected %+v, got %+v at index %d", expected7[i], item, i)
		}
	}
}
