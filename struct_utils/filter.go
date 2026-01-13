package struct_utils

import (
	"reflect"
	"strings"

	"github.com/lwy110193/go_vendor/utils"
)

// FilterOperator 定义支持的过滤操作符
type FilterOperator string

const (
	// In 包含查询
	In FilterOperator = "in"
	// NotIn 不包含查询
	NotIn FilterOperator = "not_in"
	// Prefix 前缀查询
	Prefix FilterOperator = "prefix"
	// Suffix 后缀查询
	Suffix FilterOperator = "suffix"
	// Like 模糊查询
	Like FilterOperator = "like"
	// Equal 等于
	Equal FilterOperator = "eq"
	// NotEqual 不等于
	NotEqual FilterOperator = "ne"
	// GreaterThan 大于
	GreaterThan FilterOperator = "gt"
	// GreaterThanOrEqual 大于等于
	GreaterThanOrEqual FilterOperator = "gte"
	// LessThan 小于
	LessThan FilterOperator = "lt"
	// LessThanOrEqual 小于等于
	LessThanOrEqual FilterOperator = "lte"
)

// FilterCondition 过滤条件结构体
type FilterCondition struct {
	// FieldName 结构体字段名
	FieldName string `json:"field_name"`
	// Operator 比较操作符
	Operator FilterOperator `json:"operator"`
	// Value 比较值
	Value interface{} `json:"value"`
}

// getMatchingFieldName 获取匹配的字段名，支持大小写驼峰写法匹配
func getMatchingFieldName(v reflect.Value, fieldName string) string {
	// 获取结构体类型
	typ := v.Type()
	// 遍历所有字段
	for i := 0; i < typ.NumField(); i++ {
		// 获取字段
		field := typ.Field(i)
		// 如果字段名完全匹配，直接返回
		if field.Name == fieldName {
			return field.Name
		}
		// 转换为小写驼峰进行比较，支持大小写驼峰写法
		if utils.CamelStrConv(field.Name) == utils.CamelStrConv(fieldName) {
			return field.Name
		}
	}
	// 没有匹配的字段，返回空字符串
	return ""
}

// Filter 通用结构体过滤函数
// 传入结构体对象列表和过滤条件，返回符合条件的新列表
func Filter[T any](items []T, conditions ...FilterCondition) []T {
	var result []T

	// 遍历所有结构体对象
	for _, item := range items {
		// 检查是否满足所有过滤条件
		if matchAllConditions(item, conditions) {
			result = append(result, item)
		}
	}

	return result
}

// matchAllConditions 检查结构体对象是否满足所有过滤条件
func matchAllConditions(item interface{}, conditions []FilterCondition) bool {
	// 获取结构体的反射值
	v := reflect.ValueOf(item)
	// 如果是指针，获取其指向的值
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// 确保是结构体类型
	if v.Kind() != reflect.Struct {
		return false
	}

	// 遍历所有过滤条件
	for _, condition := range conditions {
		// 获取匹配的字段名，支持大小写驼峰写法
		matchingFieldName := getMatchingFieldName(v, condition.FieldName)
		// 如果没有匹配的字段，不匹配
		if matchingFieldName == "" {
			return false
		}
		// 获取字段值
		fieldValue := v.FieldByName(matchingFieldName)

		// 检查字段值是否满足当前条件
		if !matchCondition(fieldValue, condition.Operator, condition.Value) {
			return false
		}
	}

	return true
}

// matchCondition 检查单个字段是否满足过滤条件
func matchCondition(fieldValue reflect.Value, operator FilterOperator, value interface{}) bool {
	// 获取比较值的反射值
	valueValue := reflect.ValueOf(value)

	// 根据操作符进行比较
	switch operator {
	case In:
		// 处理切片类型的Value
		if valueValue.Kind() == reflect.Slice {
			fieldVal := fieldValue.Interface()
			for i := 0; i < valueValue.Len(); i++ {
				if reflect.DeepEqual(fieldVal, valueValue.Index(i).Interface()) {
					return true
				}
			}
			return false
		}
		// 处理字符串类型的Value
		if fieldValue.Kind() == reflect.String && valueValue.Kind() == reflect.String {
			return strings.Contains(fieldValue.Interface().(string), valueValue.Interface().(string))
		}
	case NotIn:
		// 处理切片类型的Value
		if valueValue.Kind() == reflect.Slice {
			fieldVal := fieldValue.Interface()
			for i := 0; i < valueValue.Len(); i++ {
				if reflect.DeepEqual(fieldVal, valueValue.Index(i).Interface()) {
					return false
				}
			}
			return true
		}
		// 处理字符串类型的Value
		if fieldValue.Kind() == reflect.String && valueValue.Kind() == reflect.String {
			return !strings.Contains(fieldValue.Interface().(string), valueValue.Interface().(string))
		}
	case Prefix:
		if fieldValue.Kind() == reflect.String && valueValue.Kind() == reflect.String {
			return strings.HasPrefix(fieldValue.Interface().(string), valueValue.Interface().(string))
		}
	case Suffix:
		if fieldValue.Kind() == reflect.String && valueValue.Kind() == reflect.String {
			return strings.HasSuffix(fieldValue.Interface().(string), valueValue.Interface().(string))
		}
	case Like:
		if fieldValue.Kind() == reflect.String && valueValue.Kind() == reflect.String {
			return strings.Contains(fieldValue.Interface().(string), valueValue.Interface().(string))
		}
	case Equal:
		return reflect.DeepEqual(fieldValue.Interface(), value)
	case NotEqual:
		return !reflect.DeepEqual(fieldValue.Interface(), value)
	case GreaterThan:
		// 获取字段的反射类型
		fieldType := fieldValue.Type()
		// 尝试转换比较值类型
		convertedValue := reflect.New(fieldType).Elem()
		convertedValue.Set(valueValue)
		return compareValues(fieldValue, convertedValue) > 0
	case GreaterThanOrEqual:
		// 获取字段的反射类型
		fieldType := fieldValue.Type()
		// 尝试转换比较值类型
		convertedValue := reflect.New(fieldType).Elem()
		convertedValue.Set(valueValue)
		return compareValues(fieldValue, convertedValue) >= 0
	case LessThan:
		// 获取字段的反射类型
		fieldType := fieldValue.Type()
		// 尝试转换比较值类型
		convertedValue := reflect.New(fieldType).Elem()
		convertedValue.Set(valueValue)
		return compareValues(fieldValue, convertedValue) < 0
	case LessThanOrEqual:
		// 获取字段的反射类型
		fieldType := fieldValue.Type()
		// 尝试转换比较值类型
		convertedValue := reflect.New(fieldType).Elem()
		convertedValue.Set(valueValue)
		return compareValues(fieldValue, convertedValue) <= 0
	default:
		return false
	}
	return false
}

// compareValues 比较两个值的大小，返回-1（小于）、0（等于）或1（大于）
func compareValues(a, b reflect.Value) int {
	// 根据值的类型进行比较
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		va := a.Int()
		vb := b.Int()
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		va := a.Uint()
		vb := b.Uint()
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case reflect.Float32, reflect.Float64:
		va := a.Float()
		vb := b.Float()
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case reflect.String:
		va := a.String()
		vb := b.String()
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case reflect.Bool:
		va := a.Bool()
		vb := b.Bool()
		if va == vb {
			return 0
		} else if va {
			return 1
		}
		return -1
	default:
		return 0
	}
}
