package database

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/lwy110193/go_vendor/utils"
)

// whereCondition 条件运算符
var whereCondition = map[string]string{
	"GT": ">", "LT": "<", "GTE": ">=", "LTE": "<=", "EQ": "=", "NEQ": "!=",
}
var conditionList = []string{"GT", "LT", "GTE", "LTE", "EQ", "NEQ"}

// ParseWhere 拼装条件语句
func ParseWhere(where utils.MI) (whereStr string, params []interface{}) {
	whereStrBuilder := strings.Builder{}
	for field, value := range where {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(value)
			if s.Len() > 0 {
				val0 := fmt.Sprintf("%v", s.Index(0).Interface())
				if s.Len() == 2 && val0 == "LIKE" {
					whereStrBuilder.WriteString(fmt.Sprintf(" and %v like '%%%v%%'", fieldDeal(field), s.Index(1).Interface()))
				} else if s.Len() == 2 && val0 == "_STRING" {
					whereStrBuilder.WriteString(fmt.Sprintf(" and %v", s.Index(1).Interface()))
				} else if s.Len() == 2 && utils.InList(val0, conditionList) {
					whereStrBuilder.WriteString(fmt.Sprintf(" and %v %v ?", fieldDeal(field), whereCondition[val0]))
					params = append(params, s.Index(1).Interface())
				} else if s.Len() == 3 && val0 == "BETWEEN" {
					whereStrBuilder.WriteString(fmt.Sprintf(" and %v between ? and ?", fieldDeal(field)))
					params = append(params, s.Index(1).Interface(), s.Index(2).Interface())
				} else if val0 == "IN" {
					if s.Len() > 1 {
						whereStrBuilder.WriteString(fmt.Sprintf(" and %v in(", fieldDeal(field)))
						for i := 1; i < s.Len(); i++ {
							whereStrBuilder.WriteString("?,")
							params = append(params, s.Index(i).Interface())
						}
						whereStrBuilder.WriteString(")")
					}
				} else if val0 == "NOT_IN" {
					if s.Len() > 1 {
						whereStrBuilder.WriteString(fmt.Sprintf(" and %v not in(", fieldDeal(field)))
						for i := 1; i < s.Len(); i++ {
							whereStrBuilder.WriteString("?,")
							params = append(params, s.Index(i).Interface())
						}
						whereStrBuilder.WriteString(")")
					}
				} else {
					whereStrBuilder.WriteString(fmt.Sprintf(" and %v in(", fieldDeal(field)))
					for i := 0; i < s.Len(); i++ {
						whereStrBuilder.WriteString("?,")
						params = append(params, s.Index(i).Interface())
					}
					whereStrBuilder.WriteString(")")
				}
			}
		default:
			whereStrBuilder.WriteString(fmt.Sprintf(" and %v = ?", fieldDeal(field)))
			params = append(params, value)
		}
	}
	if whereStrBuilder.Len() > 0 {
		whereStr = whereStrBuilder.String()[4:]
	} else if whereStrBuilder.Len() == 0 {
		whereStr = " 1=1 "
	}
	return
}

// fieldDeal 字段处理
func fieldDeal(field string) string {
	if strings.Contains(field, ".") {
		return fmt.Sprintf("`%v`", field)
	} else {
		return field
	}
}

// ParseDateWhere 日期范围处理
func ParseDateWhere(date string, endOfDay bool) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	if endOfDay {
		t = t.Add(time.Hour*24 - time.Second)
	}
	return t.Format("2006-01-02 15:04:05.999")
}

// ParsePage 分页处理
func ParsePage(pageSize, page int) string {
	return fmt.Sprintf(" limit %v,%v", utils.Max((page-1)*pageSize, 0), pageSize)
}

// DbExtInfo 数据库扩展信息
type DbExtInfo struct {
	PageInfo  *PageInfo  `json:"page_info"`
	OrderInfo *OrderInfo `json:"order_info"`
}

// OrderInfo 排序信息
type OrderInfo struct {
	Field     string `json:"field"`      // 排序字段
	OrderType string `json:"order_type"` // 排序类型：asc正序，desc倒序
}

// PageInfo 分页信息
type PageInfo struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
