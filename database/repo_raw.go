package database

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"gitee.com/qq1101931365/go_verdor/utils"
	perrors "github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	
)

// Raw 原始SQL查询
func (r *BaseRepo) Raw(ctx context.Context, result interface{}, sql string, params ...interface{}) (err error) {
	err = r.Db.WithContext(ctx).Raw(sql, params...).Scan(result).Error
	if err != nil {
		return err
	}
	return nil
}

// Exec 执行原始SQL语句
func (r *BaseRepo) Exec(ctx context.Context, sql string, params ...interface{}) (err error) {
	err = r.Db.WithContext(ctx).Exec(sql, params...).Error
	if err != nil {
		return err
	}
	return nil
}

// Transaction 事务处理
func (r *BaseRepo) Transaction(ctx context.Context, fun func(tx *gorm.DB) error) (err error) {
	err = r.Db.WithContext(ctx).Transaction(fun)
	if err != nil {
		return perrors.WithStack(err)
	}
	return nil
}

// UpdateOrInsert 更新或插入；更新数量为0时插入，需指定更新条件字段
func (r *BaseRepo) UpdateOrInsert(ctx context.Context, data schema.Tabler, updateWhereField []string, ignoreUpdateField []string) (err error) {
	if data.TableName() == "" || len(updateWhereField) == 0 {
		return
	}

	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}

	nowTime := time.Now()
	baseInfo := BaseModel{}
	var insertFieldList, insertPlaceHolder []string
	var insertParams, updateParams, updateWhereParams []interface{}
	var updateSetStr, updateWhereStr string
	for i := 0; i < dataType.NumField(); i++ {
		dbField := utils.CamelStrConv(dataType.Field(i).Name)
		if dbField == "base_model" {
			baseDataType := reflect.TypeOf(dataValue.Field(i).Interface())
			baseDataValue := reflect.ValueOf(dataValue.Field(i).Interface())
			if baseDataType.Kind() == reflect.Ptr {
				baseDataType = dataType.Elem()
				baseDataValue = dataValue.Elem()
			}
			baseInfo.ID = baseDataValue.FieldByName("ID").Interface().(uint64)
			baseInfo.CreatedAt = baseDataValue.FieldByName("CreatedAt").Interface().(time.Time)
			baseInfo.UpdatedAt = baseDataValue.FieldByName("UpdatedAt").Interface().(time.Time)
			baseInfo.DeletedAt = baseDataValue.FieldByName("DeletedAt").Interface().(time.Time)

			// if utils.InList("id", updateWhereField) && baseInfo.ID > 0 {
			// 	updateWhereStr += "`id`=? and "
			// 	updateWhereParams = append(updateWhereParams, baseInfo.ID)
			// }
			continue
		} else {
			value := dataValue.Field(i).Interface()
			if dbField == "date" && utils.InList(dataType.Field(i).Type.String(), []string{"time.Time", "*time.Time"}) {
				if dataType.Field(i).Type.String() == "time.Time" {
					value = value.(time.Time).Format(time.DateOnly)
				} else {
					value = value.(*time.Time).Format(time.DateOnly)
				}
			}
			insertFieldList = append(insertFieldList, fmt.Sprintf("`%v`", dbField))
			insertPlaceHolder = append(insertPlaceHolder, "?")
			insertParams = append(insertParams, value)
			if !utils.InList(dbField, ignoreUpdateField) {
				updateSetStr += fmt.Sprintf("`%v`=?,", dbField)
				updateParams = append(updateParams, value)
			}
			if utils.InList(dbField, updateWhereField) {
				updateWhereStr += fmt.Sprintf("`%v`=? and ", dbField)
				updateWhereParams = append(updateWhereParams, value)
			}
		}
	}
	if len(updateWhereParams) == 0 {
		return errors.New("UpdateOrInsert where params err")
	}

	insertFieldList = append(insertFieldList, "`created_at`", "`updated_at`")
	insertPlaceHolder = append(insertPlaceHolder, "?", "?")

	if !baseInfo.CreatedAt.IsZero() { // .created_at 不为空时，更新时包含该字段
		updateSetStr += fmt.Sprintf("`%v`=?,", "created_at")
		updateParams = append(updateParams, baseInfo.CreatedAt)

		insertParams = append(insertParams, baseInfo.CreatedAt)
	} else {
		insertParams = append(insertParams, &nowTime)
	}

	updateSetStr += fmt.Sprintf("`%v`=?,", "updated_at")
	if !baseInfo.UpdatedAt.IsZero() {
		updateParams = append(updateParams, baseInfo.UpdatedAt)
		insertParams = append(insertParams, baseInfo.UpdatedAt)
	} else {
		updateParams = append(updateParams, &nowTime)
		insertParams = append(insertParams, &nowTime)
	}

	if !baseInfo.DeletedAt.IsZero() {
		updateSetStr += fmt.Sprintf("`%v`=?,", "deleted_at")
		updateParams = append(updateParams, baseInfo.DeletedAt)

		insertPlaceHolder = append(insertPlaceHolder, "?")
		insertFieldList = append(insertFieldList, "`deleted_at`")
		insertParams = append(insertParams, baseInfo.DeletedAt)
	}
	if baseInfo.ID > 0 {
		insertPlaceHolder = append(insertPlaceHolder, "?")
		insertFieldList = append(insertFieldList, "`id`")
		insertParams = append(insertParams, baseInfo.ID)
	}

	needInsert := false
	updateSql := fmt.Sprintf("update %v set %v where %v", data.TableName(), updateSetStr[:len(updateSetStr)-1], updateWhereStr[:len(updateWhereStr)-4])
	tmp := r.Db.WithContext(ctx).Exec(updateSql, append(updateParams, updateWhereParams...)...)
	if err = tmp.Error; err != nil {
		return
	}
	if tmp.RowsAffected == 0 {
		needInsert = true
	}
	if needInsert {
		insertSql := fmt.Sprintf("insert into %v(%v) values(%v)", data.TableName(), strings.Join(insertFieldList, ","), strings.Join(insertPlaceHolder, ","))
		if err = r.Db.WithContext(ctx).Exec(insertSql, insertParams...).Error; err != nil {
			return
		}
	}
	return
}

type CaseWhenThen struct {
	When utils.MI
	Then interface{}
}

// UpdateInBatchForStruct 批量更新数据
// caseWhenField 是需要进行case when then 处理的字段
// ignoreUpdateField 是不需要进行更新的字段
func (r *BaseRepo) UpdateInBatchForStruct(ctx context.Context, tableName string, list []interface{}, where utils.MI, caseWhenField []string, ignoreUpdateField []string) (err error) {
	var dataList []utils.MI
	for _, item := range list {
		itemMapInfo := utils.MI{}
		utils.ConvStructToMap(item, itemMapInfo)
		dataList = append(dataList, itemMapInfo)
	}

	return r.UpdateInBatchForMap(ctx, tableName, dataList, where, caseWhenField, ignoreUpdateField)
}

// UpdateInBatchForMap 批量更新数据
// caseWhenField 是需要进行case when then 处理的字段
// ignoreUpdateField 是不需要进行更新的字段
func (r *BaseRepo) UpdateInBatchForMap(ctx context.Context, tableName string, dataList []utils.MI, where utils.MI, caseWhenField []string, ignoreUpdateField []string) (err error) {
	if len(dataList) == 0 {
		return
	}
	if len(caseWhenField) == 0 {
		return errors.New("caseWhenField error")
	}

	ignoreUpdateField = append(ignoreUpdateField, "id", "created_at") // id不进行更新
	whereStr, WhereParams := ParseWhere(where)

	nowTime := time.Now()
	for {
		uptList := utils.Truncate(&dataList, 40)
		if len(uptList) == 0 {
			break
		}

		fieldCaseMap := map[string][]*CaseWhenThen{}
		for _, item := range uptList {
			if val, ok := item["updated_at"]; ok {
				if reflect.TypeOf(val).String() == "*time.Time" && val.(*time.Time) == nil {
					item["updated_at"] = &nowTime
				} else if reflect.TypeOf(val).String() == "time.Time" && val.(time.Time).IsZero() {
					item["updated_at"] = nowTime
				}
			}
			caseWhenItem := CaseWhenThen{When: map[string]interface{}{}}
			for _, field := range caseWhenField {
				if value, ok := item[field]; ok {
					if field == "date" {
						rfType := reflect.TypeOf(value)
						if utils.InList(rfType.String(), []string{"time.Time", "*time.Time"}) {
							if rfType.String() == "time.Time" {
								value = value.(time.Time).Format(time.DateOnly)
							} else {
								value = value.(*time.Time).Format(time.DateOnly)
							}
						}
					}
					caseWhenItem.When[field] = value
				} else {
					return fmt.Errorf("caseWhenField %v not exist", field)
				}
			}
			for field, value := range item {
				if !utils.InList(field, caseWhenField) && !utils.InList(field, ignoreUpdateField) {
					tmpCaseWhen := caseWhenItem
					tmpCaseWhen.Then = value
					fieldCaseMap[field] = append(fieldCaseMap[field], &tmpCaseWhen)
				}
			}
		}

		_sql := fmt.Sprintf("update `%v` set", tableName)
		var params []interface{}
		for field, caseList := range fieldCaseMap {
			_sql += fmt.Sprintf(" `%v`= case", field)
			for _, caseItem := range caseList {
				tmpWhereStr, tmpParams := ParseWhere(caseItem.When)
				_sql += fmt.Sprintf(" when %v then ?", tmpWhereStr)
				params = append(params, tmpParams...)
				params = append(params, caseItem.Then)
			}
			_sql += fmt.Sprintf(" else `%v` end,", field)
		}
		params = append(params, WhereParams...)
		_sql = _sql[:len(_sql)-1] + fmt.Sprintf(" where %v", whereStr)
		if err = r.Exec(ctx, _sql, params...); err != nil {
			return
		}
	}
	return
}
