package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// CamelStrConv 驼峰字符串 转 下划线字符串
func CamelStrConv(str string) string {
	// 处理特殊情况
	switch str {
	case "ID":
		return "id"
	case "UpdatedAt":
		return "updated_at"
	case "CreatedAt":
		return "created_at"
	}

	// 使用strings.Builder优化字符串拼接性能
	var builder strings.Builder
	builder.Grow(len(str) * 2) // 预分配足够的空间，最多可能翻倍长度

	for i, c := range str {
		// 直接判断大写字母范围，避免map查找
		if c >= 'A' && c <= 'Z' {
			// 不是第一个字符时添加下划线
			if i > 0 {
				builder.WriteByte('_')
			}
			// 转换为小写字母
			builder.WriteRune(c + 32)
		} else {
			builder.WriteRune(c)
		}
	}

	// 构建结果字符串
	result := builder.String()

	// 处理特殊情况：如果结果以_开头，去掉开头的_
	if len(result) > 0 && result[0] == '_' {
		return result[1:]
	}

	return result
}

// SetAttrValue 设置结构体字段值，字段传值可 驼峰 或 下划线字符串
func SetAttrValue(data interface{}, field string, value interface{}) {
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}

	for i := 0; i < dataType.NumField(); i++ {
		if dataType.Field(i).Name == field || CamelStrConv(dataType.Field(i).Name) == field {
			dataValue.Field(i).Set(reflect.ValueOf(value))
			return
		}
	}
}

// 获取切片结构体中的指定key值的切片
func GetAttrValueList(data interface{}, field string, result interface{}) {
	// 处理data参数
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}
	if dataType.Kind() != reflect.Slice {
		return
	}

	// 处理result参数
	resultType := reflect.TypeOf(result)
	resultValue := reflect.ValueOf(result)
	if resultType.Kind() != reflect.Ptr {
		return
	}
	resultType = resultType.Elem()
	resultValue = resultValue.Elem()
	if resultType.Kind() != reflect.Slice {
		return
	}

	// 创建新的结果切片
	resultSlice := reflect.MakeSlice(resultType, dataValue.Len(), dataValue.Len())

	for i := 0; i < dataValue.Len(); i++ {
		itemValue := dataValue.Index(i)
		if itemValue.Kind() == reflect.Ptr {
			itemValue = itemValue.Elem()
		}
		if itemValue.Kind() != reflect.Struct {
			continue
		}

		// 查找字段
		var fieldValue reflect.Value
		itemType := itemValue.Type()
		found := false

		for j := 0; j < itemType.NumField(); j++ {
			structField := itemType.Field(j)

			// 检查字段名
			if structField.Name == field {
				fieldValue = itemValue.Field(j)
				found = true
				break
			}

			// 检查驼峰转换后的字段名
			if CamelStrConv(structField.Name) == field {
				fieldValue = itemValue.Field(j)
				found = true
				break
			}

			// 检查JSON标签
			jsonTag := structField.Tag.Get("json")
			if jsonTag != "" {
				// 提取JSON标签中的字段名（忽略选项）
				tagName := jsonTag
				if idx := strings.Index(jsonTag, ","); idx > 0 {
					tagName = jsonTag[:idx]
				}
				if tagName == field {
					fieldValue = itemValue.Field(j)
					found = true
					break
				}
			}
		}

		// 如果找到字段，设置到结果切片中
		if found && fieldValue.IsValid() {
			resultItem := resultSlice.Index(i)
			if resultItem.CanSet() {
				// 尝试设置值，处理类型转换
				if fieldValue.Type().AssignableTo(resultItem.Type()) {
					resultItem.Set(fieldValue)
				} else {
					// 尝试使用反射转换类型
					trySetValue(resultItem, fieldValue)
				}
			}
		}
	}

	// 将结果设置回result参数
	resultValue.Set(resultSlice)
}

// trySetValue 尝试将一个值设置到另一个值，处理类型转换
func trySetValue(dst, src reflect.Value) {
	if !dst.CanSet() {
		return
	}

	// 如果类型相同，直接设置
	if src.Type().AssignableTo(dst.Type()) {
		dst.Set(src)
		return
	}

	// 尝试基本类型转换
	switch dst.Kind() {
	case reflect.String:
		if src.Kind() == reflect.String {
			dst.SetString(src.String())
		} else {
			dst.SetString(fmt.Sprintf("%v", src.Interface()))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if src.Kind() >= reflect.Int && src.Kind() <= reflect.Int64 {
			dst.SetInt(src.Int())
		} else if src.Kind() >= reflect.Uint && src.Kind() <= reflect.Uint64 {
			dst.SetInt(int64(src.Uint()))
		} else if src.Kind() >= reflect.Float32 && src.Kind() <= reflect.Float64 {
			dst.SetInt(int64(src.Float()))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if src.Kind() >= reflect.Uint && src.Kind() <= reflect.Uint64 {
			dst.SetUint(src.Uint())
		} else if src.Kind() >= reflect.Int && src.Kind() <= reflect.Int64 {
			dst.SetUint(uint64(src.Int()))
		} else if src.Kind() >= reflect.Float32 && src.Kind() <= reflect.Float64 {
			dst.SetUint(uint64(src.Float()))
		}
	case reflect.Float32, reflect.Float64:
		if src.Kind() >= reflect.Float32 && src.Kind() <= reflect.Float64 {
			dst.SetFloat(src.Float())
		} else if src.Kind() >= reflect.Int && src.Kind() <= reflect.Int64 {
			dst.SetFloat(float64(src.Int()))
		} else if src.Kind() >= reflect.Uint && src.Kind() <= reflect.Uint64 {
			dst.SetFloat(float64(src.Uint()))
		}
	case reflect.Bool:
		if src.Kind() == reflect.Bool {
			dst.SetBool(src.Bool())
		}
	}
}

// DataConvert 复制不同struct同key值到另一个结构体
func DataConvert(from interface{}, to interface{}) {
	typeOfFrom := reflect.TypeOf(from)
	valueOfFrom := reflect.ValueOf(from)
	if typeOfFrom.Kind() == reflect.Ptr {
		typeOfFrom = typeOfFrom.Elem()
		valueOfFrom = valueOfFrom.Elem()
	}
	fromFieldTypeMap := map[string]string{}
	fromFieldValueMap := map[string]interface{}{}
	for i := 0; i < typeOfFrom.NumField(); i++ {
		fromFieldTypeMap[typeOfFrom.Field(i).Name] = typeOfFrom.Field(i).Type.String()
		fromFieldValueMap[typeOfFrom.Field(i).Name] = valueOfFrom.Field(i).Interface()
	}

	typeOfTo := reflect.TypeOf(to)
	valueOfTo := reflect.ValueOf(to)
	if typeOfTo.Kind() == reflect.Ptr {
		typeOfTo = typeOfTo.Elem()
		valueOfTo = valueOfTo.Elem()
	}
	for i := 0; i < typeOfTo.NumField(); i++ {
		if fromFieldValueMap[typeOfTo.Field(i).Name] != nil && fromFieldTypeMap[typeOfTo.Field(i).Name] == typeOfTo.Field(i).Type.String() && valueOfTo.Field(i).CanSet() {
			valueOfTo.Field(i).Set(reflect.ValueOf(fromFieldValueMap[typeOfTo.Field(i).Name]))
		}
	}

}

// ConvStructToMap 结构体转map，只处理一级
func ConvStructToMap(data interface{}, result MI) {
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}

	for i := 0; i < dataType.NumField(); i++ {
		field := CamelStrConv(dataType.Field(i).Name)
		value := dataValue.Field(i).Interface()
		if reflect.TypeOf(value).Kind() == reflect.Struct && dataType.Field(i).Anonymous {
			ConvStructToMap(value, result)
		} else {
			result[field] = value
		}
	}
}

// ConvStructToGetParam 结构体转GET参数字符串，只处理一级
func ConvStructToGetParam(data interface{}, str_ptr *string) {
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}

	for i := 0; i < dataType.NumField(); i++ {
		field := CamelStrConv(dataType.Field(i).Name)
		value := dataValue.Field(i).Interface()
		if reflect.TypeOf(value).Kind() == reflect.Struct && dataType.Field(i).Anonymous {
			ConvStructToGetParam(value, str_ptr)
		} else {
			*str_ptr += fmt.Sprintf("%s=%v&", field, value)
		}
	}
}

// ConvStructToGetParamFromJsonTag 结构体转GET参数字符串，只处理一级，字段名从json标签获取
func ConvStructToGetParamFromJsonTag(data interface{}, str_ptr *string) {
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}

	for i := 0; i < dataType.NumField(); i++ {
		field := dataType.Field(i).Tag.Get("json")
		if field == "" {
			continue
		}
		value := dataValue.Field(i).Interface()
		if reflect.TypeOf(value).Kind() == reflect.Struct && dataType.Field(i).Anonymous {
			ConvStructToGetParam(value, str_ptr)
		} else {
			*str_ptr += fmt.Sprintf("%s=%v&", field, value)
		}
	}
}
