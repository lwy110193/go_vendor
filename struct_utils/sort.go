package struct_utils

import (
	"reflect"
	"sort"
	"strconv"
)

// SortDirection 定义排序方向
type SortDirection string

const (
	// Ascending 升序
	Ascending SortDirection = "asc"
	// Descending 降序
	Descending SortDirection = "desc"
)

// SortConfig 排序配置结构体
type SortConfig struct {
	// FieldName 结构体字段名
	FieldName string `json:"field_name"`
	// Direction 排序方向
	Direction SortDirection `json:"direction"`
	// ParseStringAsNumber 是否将字符串转换为数值型进行比较
	ParseStringAsNumber bool `json:"parse_string_as_number"`
}

// Sort 通用结构体排序函数
// 传入结构体对象列表和排序配置，返回排序后的新列表
func Sort[T any](items []T, configs ...SortConfig) []T {
	// 创建切片副本，避免修改原切片
	result := make([]T, len(items))
	copy(result, items)

	// 使用sort.Slice进行排序
	sort.Slice(result, func(i, j int) bool {
		return compareItems(result[i], result[j], configs)
	})

	return result
}

// compareItems 比较两个结构体对象
func compareItems(a, b interface{}, configs []SortConfig) bool {
	// 获取结构体的反射值
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	// 如果是指针，获取其指向的值
	if va.Kind() == reflect.Ptr {
		va = va.Elem()
	}
	if vb.Kind() == reflect.Ptr {
		vb = vb.Elem()
	}

	// 确保是结构体类型
	if va.Kind() != reflect.Struct || vb.Kind() != reflect.Struct {
		return false
	}

	// 遍历所有排序配置
	for _, config := range configs {
		// 获取匹配的字段名
		matchingFieldName := getMatchingFieldName(va, config.FieldName)
		// 如果没有匹配的字段，跳过该排序配置
		if matchingFieldName == "" {
			continue
		}
		// 获取字段值
		fieldA := va.FieldByName(matchingFieldName)
		fieldB := vb.FieldByName(matchingFieldName)

		// 比较字段值
		result := compareFieldValues(fieldA, fieldB, config.ParseStringAsNumber)

		// 根据排序方向返回结果
		if result != 0 {
			if config.Direction == Ascending {
				return result < 0
			}
			return result > 0
		}
	}

	// 所有配置都相等，保持原顺序
	return false
}

// compareFieldValues 比较两个字段值
func compareFieldValues(a, b reflect.Value, parseStringAsNumber bool) int {
	// 如果需要将字符串转换为数值型比较
	if parseStringAsNumber && a.Kind() == reflect.String && b.Kind() == reflect.String {
		// 尝试将字符串转换为float64
		aNum, errA := strconv.ParseFloat(a.String(), 64)
		bNum, errB := strconv.ParseFloat(b.String(), 64)

		// 如果两个字符串都能转换为数值，按数值比较
		if errA == nil && errB == nil {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// 否则使用原有比较逻辑
	return compareValues(a, b)
}
