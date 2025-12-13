package inject

import (
	"errors"
	"reflect"

	"go.uber.org/dig"
)

var digContainer = dig.New()

func GetContainer() *dig.Container {
	return digContainer
}

func Provide(constructor any, opts ...dig.ProvideOption) error {
	return digContainer.Provide(constructor, opts...)
}

func Invoke(invokeFn any, opts ...dig.InvokeOption) error {
	return digContainer.Invoke(invokeFn, opts...)
}

func Resolve(result any) error {
	// 检查result是否为指针
	if reflect.TypeOf(result).Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

	// 获取目标类型
	targetType := reflect.TypeOf(result).Elem()

	// 创建一个函数，该函数接受目标类型作为参数
	// 然后使用反射将解析的值设置到目标指针
	setFn := reflect.MakeFunc(
		reflect.FuncOf(
			[]reflect.Type{targetType}, // 输入参数类型
			[]reflect.Type{},           // 返回值类型
			false,                      // 不支持可变参数
		),
		func(args []reflect.Value) []reflect.Value {
			// 获取目标值（指针的元素）
			targetValue := reflect.ValueOf(result).Elem()
			// 设置解析的值
			targetValue.Set(args[0])
			return []reflect.Value{}
		},
	)

	// 调用创建的函数，dig会自动解析参数
	return digContainer.Invoke(setFn.Interface())
}
