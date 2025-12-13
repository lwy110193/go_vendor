package database

import (
	"context"
	"fmt"

	"gitee.com/qq1101931365/go_verdor/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type BaseRepo struct {
	Db    *gorm.DB
	Model schema.Tabler
}

// Find 查找数据
func (r *BaseRepo) Find(ctx context.Context, where utils.MI, info *DbExtInfo, fieldList ...string) (list []schema.Tabler, cnt int64, err error) {
	db := r.Db.WithContext(ctx)
	query, args := ParseWhere(where)
	if len(fieldList) > 0 {
		db = db.Select(fieldList)
	}
	if len(query) > 0 {
		db = db.Where(query, args...)
	}

	if info != nil {
		if info.PageInfo != nil {
			db = db.Model(r.Model).Count(&cnt).Offset(utils.Max((info.PageInfo.Page-1)*info.PageInfo.PageSize, 0)).Limit(info.PageInfo.PageSize)
		}
		if info.OrderInfo != nil {
			db = db.Order(fmt.Sprintf("%v %v", info.OrderInfo.Field, info.OrderInfo.OrderType))
		}
	}
	if err = db.Find(&list).Error; err != nil {
		return nil, 0, errors.WithStack(err)
	}

	return list, cnt, nil
}

// FindOne 查找一条数据
func (r *BaseRepo) FindOne(ctx context.Context, where utils.MI, fieldList ...string) (schema.Tabler, error) {
	db := r.Db.WithContext(ctx)
	query, args := ParseWhere(where)
	if len(fieldList) > 0 {
		db = db.Select(fieldList)
	}
	if len(query) > 0 {
		db = db.Where(query, args...)
	}

	if err := db.First(r.Model).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return r.Model, nil
}

// Create 创建一条数据
func (r *BaseRepo) Create(ctx context.Context, data schema.Tabler) error {
	if err := r.Db.WithContext(ctx).Create(data).Error; err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// CreateBatch 创建多条数据
func (r *BaseRepo) CreateBatch(ctx context.Context, list []schema.Tabler, batchSize int) error {
	if err := r.Db.WithContext(ctx).CreateInBatches(list, batchSize).Error; err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Update 更新数据 - 通过map更新数据
func (r *BaseRepo) Update(ctx context.Context, where, upt utils.MI) error {
	db := r.Db.WithContext(ctx)
	query, args := ParseWhere(where)
	if len(query) > 0 {
		db = db.Where(query, args...)
	}
	if err := db.Model(r.Model).Where(where).Updates(upt).Error; err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Updates 更新数据 - 通过对象更新数据 - 更新对象中的非零值字段
func (r *BaseRepo) Updates(ctx context.Context, data schema.Tabler, where utils.MI) (err error) {
	whereStr, params := ParseWhere(where)
	err = r.Db.WithContext(ctx).Model(data).Where(whereStr, params...).Updates(data).Error
	return
}

// UpdatesWithZeroValue 更新数据 - 通过对象更新数据 - 更新对象中全部字段
func (r *BaseRepo) UpdatesWithZeroValue(ctx context.Context, data schema.Tabler, where utils.MI, ignoreFields ...string) (err error) {
	if len(ignoreFields) == 0 {
		ignoreFields = append(ignoreFields, "id", "created_at")
	}
	whereStr, params := ParseWhere(where)
	mapData := utils.MI{}
	utils.ConvStructToMap(data, mapData)
	for field, _ := range mapData {
		if utils.InList(field, ignoreFields) {
			delete(mapData, field)
		}
	}
	err = r.Db.WithContext(ctx).Model(data).Where(whereStr, params...).Updates(mapData).Error
	return
}

// Delete 删除数据
func (r *BaseRepo) Delete(ctx context.Context, where utils.MI) error {
	db := r.Db.WithContext(ctx)
	query, args := ParseWhere(where)
	if len(query) > 0 {
		db = db.Where(query, args...)
	}
	if err := db.Delete(r.Model).Error; err != nil {
		return errors.WithStack(err)
	}
	return nil
}
