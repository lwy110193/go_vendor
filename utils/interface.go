package utils

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// MI 通用map
type MI map[string]interface{}

// AddToUniqueList 添加到列表（去重）
func AddToUniqueList[T comparable](list *[]T, addItem T) {
	for _, item := range *list {
		if item == addItem {
			return
		}
	}
	*list = append(*list, addItem)
}

// Max 获取最大值
func Max[T int | int8 | int16 | int32 | int64 | float32 | float64](nums ...T) T {
	var maxNum T
	for _, num := range nums {
		if num > maxNum {
			maxNum = num
		}
	}
	return maxNum
}

// InList 查询是否在列表
func InList[T any](data T, list []T) bool {
	if len(list) == 0 {
		return false
	}
	for _, item := range list {
		if fmt.Sprintf("%v", data) == fmt.Sprintf("%v", item) {
			return true
		}
	}
	return false
}

// RandNumCode 指定位数随机数
func RandNumCode(num int) (code string) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < num; i++ {
		code += fmt.Sprintf("%v", rand.Int()%10)
	}
	return
}

// RetainTwoPoint 保留两位小数
func RetainTwoPoint(num float64) float64 {
	roundNum := math.Round(num*100) / 100
	return roundNum
}

// Truncate 截取一段切片，原切片将缩短
func Truncate[T any](list *[]T, length int) (tList []T) {
	if list == nil || len(*list) == 0 {
		return
	}
	if len(*list) > length {
		tList = (*list)[:length]
		*list = (*list)[length:]
	} else {
		tList = *list
		*list = []T{}
	}
	return
}

var m = map[string]string{
	"A": "_a",
	"B": "_b",
	"C": "_c",
	"D": "_d",
	"E": "_e",
	"F": "_f",
	"G": "_g",
	"H": "_h",
	"I": "_i",
	"J": "_j",
	"K": "_k",
	"L": "_l",
	"M": "_m",
	"N": "_n",
	"O": "_o",
	"P": "_p",
	"Q": "_q",
	"R": "_r",
	"S": "_s",
	"T": "_t",
	"U": "_u",
	"V": "_v",
	"W": "_w",
	"X": "_x",
	"Y": "_y",
	"Z": "_z",
}

// CamelStrConv 驼峰字符串 转 下划线字符串
func CamelStrConv(str string) (newStr string) {
	//re := regexp.MustCompile("([a-z0-9])([A-Z])")
	//return strings.ToLower(re.ReplaceAllString(str, "${1}_${2}"))

	//re := re2.MustCompile("([a-z0-9])([A-Z])")
	//return strings.ToLower(re.ReplaceAllString(str, "${1}_${2}"))

	if str == "ID" {
		return "id"
	} else if str == "UpdatedAt" {
		return "updated_at"
	} else if str == "CreatedAt" {
		return "created_at"
	}

	var tmp string
	for _, c := range str {
		tmp = string(c)
		if r, ok := m[tmp]; ok {
			newStr += r
		} else {
			newStr += tmp
		}
	}
	if len(newStr) > 0 && string(newStr[0]) == "_" {
		newStr = newStr[1:]
	}
	return
}

// ToInterfaceSlice 转换为interface切片
func ToInterfaceSlice[T any](list []T) (data []interface{}) {
	for _, item := range list {
		data = append(data, item)
	}
	return
}

// ConvToInt64 字符串转int64，失败返回0
func ConvToInt64(str string) int64 {
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// StringToInt 字符串转int，失败返回0
func StringToInt(data string) int {
	val, err := strconv.Atoi(data)
	if err != nil {
		return 0
	}
	return val
}

// StringToInt32 字符串转int32，失败返回0
func StringToInt32(data string) int32 {
	val, err := strconv.ParseInt(data, 10, 32)
	if err != nil {
		return 0
	}
	return int32(val)
}

// StringToFloat64 字符串转float64，失败返回0
func StringToFloat64(data string) float64 {
	val, err := strconv.ParseFloat(data, 64)
	if err != nil {
		return 0
	}
	return val
}

// Float64Add 浮点加，处理进度丢失
func Float64Add(a, b float64) float64 {
	var v1 = decimal.NewFromFloat(a)
	var v2 = decimal.NewFromFloat(b)
	var v3 = v1.Add(v2)
	fVal, _ := v3.Float64()
	return fVal
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

// ConvStructToMap 结构体转map（包含内嵌结构体处理）
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

// RangeDateList 开始日期到结束日期之间日期列表
func RangeDateList(startDate string, endDate string) (list []string) {
	if len(startDate) > 10 {
		startDate = startDate[:10]
	}
	if len(endDate) > 10 {
		endDate = endDate[:10]
	}
	sd, err := time.Parse(time.DateOnly, startDate)
	if err != nil {
		return
	}
	ed, err := time.Parse(time.DateOnly, endDate)
	if err != nil {
		return
	}
	days := (int)(ed.Unix()-sd.Unix()) / (24 * 60 * 60)
	for i := 0; i <= days; i++ {
		list = append(list, sd.AddDate(0, 0, i).Format(time.DateOnly))
	}
	return
}

// 获取指定日期所在周的开始日期（周一）
func getWeekStartDate(date time.Time) time.Time {
	offset := int(time.Monday - date.Weekday())
	if offset > 0 {
		offset = -6
	}
	return date.AddDate(0, 0, offset)
}

// 获取指定日期所在周的结束日期（周日）
func getWeekEndDate(date time.Time) time.Time {
	return getWeekStartDate(date).AddDate(0, 0, 6)
}

// 获取指定日期是当年的第几周
func getWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

// RangeWeekList 开始日期到结束日期之间周列表,以及开始时间所在周的第一天，以及结束时间所在周的最后一天
func RangeWeekList(startDate string, endDate string) (list []string, sDate, eDate string) {
	sd, err := time.Parse(time.DateOnly, startDate)
	if err != nil {
		return
	}
	ed, err := time.Parse(time.DateOnly, endDate)
	if err != nil {
		return
	}
	if sd.Unix() > ed.Unix() {
		return
	}
	startDateOfFirstWeek := getWeekStartDate(sd)
	sDate = startDateOfFirstWeek.Format(time.DateOnly)
	endDateOfLastWeek := getWeekEndDate(ed)
	eDate = endDateOfLastWeek.Format(time.DateOnly)

	for tmp := startDateOfFirstWeek; tmp.Before(endDateOfLastWeek); tmp = tmp.AddDate(0, 0, 7) {
		weekStr := tmp.Format("2006-")
		weekNumber := getWeekNumber(tmp)
		if weekNumber < 10 {
			weekStr += fmt.Sprintf("0%d", weekNumber)
		} else {
			weekStr += fmt.Sprintf("%d", weekNumber)
		}
		list = append(list, weekStr)
	}

	return
}

// ParseWeek 将周转换为日期范围
func ParseWeek(weekStr string) (str string) {
	tmp := strings.Split(weekStr, "-")
	if len(tmp) != 2 || len(tmp[0]) != 4 || len(tmp[1]) != 2 {
		return
	}

	var week int
	year := StringToInt(tmp[0])
	if tmp[0] == "0" {
		week = StringToInt(tmp[1][1:])
	} else {
		week = StringToInt(tmp[1])
	}

	firstDayOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	str = firstDayOfYear.AddDate(0, 0, (week-1)*7).Format(time.DateOnly)
	str += " - " + firstDayOfYear.AddDate(0, 0, (week)*7-1).Format(time.DateOnly)
	return
}

// RangeMonthList 开始日期到结束日期之间月列表,以及开始时间所在月的第一天，以及结束时间所在月的最后一天
func RangeMonthList(startDate string, endDate string) (list []string, sDate, eDate string) {
	sd, err := time.Parse(time.DateOnly, startDate)
	if err != nil {
		return
	}
	ed, err := time.Parse(time.DateOnly, endDate)
	if err != nil {
		return
	}
	if sd.Unix() > ed.Unix() {
		return
	}

	startDateOfFirstMonth := time.Date(sd.Year(), sd.Month(), 1, 0, 0, 0, 0, sd.Location())
	sDate = startDateOfFirstMonth.Format(time.DateOnly)

	nextMonth := ed.Month() + 1
	year := ed.Year()
	if nextMonth > 12 {
		nextMonth = 1
		year++
	}
	endDateOfLastMonth := time.Date(year, nextMonth, 1, 0, 0, 0, 0, ed.Location()).Add(-24 * time.Hour)
	eDate = endDateOfLastMonth.Format(time.DateOnly)
	startDateOfLastMonth := time.Date(ed.Year(), ed.Month(), 1, 0, 0, 0, 0, sd.Location())

	for tmp := startDateOfFirstMonth; tmp.Before(startDateOfLastMonth.AddDate(0, 0, 1)); tmp = tmp.AddDate(0, 1, 0) {
		monthStr := tmp.Format("2006-01")
		list = append(list, monthStr)
	}

	return
}

// FormatTimeToZero 时间转换到0时区
func FormatTimeToZero(targetTime string, timezone string) (string, error) {
	if targetTime == "" || timezone == "" {
		return "", nil
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", err
	}
	t, _ := time.ParseInLocation(time.DateTime, targetTime, loc)
	return t.In(time.UTC).Format(time.DateTime), nil
}

// FormatTimeToLocal 时间转换到本地时区
func FormatTimeToLocal(targetTime string, timezone string) (string, error) {
	if targetTime == "" || timezone == "" {
		return "", nil
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", err
	}
	t, _ := time.ParseInLocation(time.DateTime, targetTime, time.UTC)
	return t.In(loc).Format(time.DateTime), nil
}

// FormatTimeToSpecifyTimezone 时间转换到指定时区
func FormatTimeToSpecifyTimezone(targetTime string, fromTimezone, toTimezone string) (string, error) {
	if targetTime == "" || fromTimezone == "" || toTimezone == "" {
		return "", nil
	}
	loc, err := time.LoadLocation(fromTimezone)
	if err != nil {
		return "", err
	}
	t, err := time.ParseInLocation(time.DateTime, targetTime, loc)
	if err != nil {
		return "", err
	}

	targetLoc, err := time.LoadLocation(toTimezone)
	if err != nil {
		return "", err
	}
	newT := t.In(targetLoc)
	return newT.Format(time.DateTime), nil
}
