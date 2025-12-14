package database_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lwy110193/go_vendor/database"
	"github.com/lwy110193/go_vendor/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		"root",
		"mysql_8j5rrb",
		"192.168.3.42",
		3306,
		"stock",
	)

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: nil,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MySQL database: %v", err))
	}

	// 获取底层的sql.DB对象进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("Failed to get underlying sql.DB: %v", err))
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
}

// StockInfo 表示股票信息表
type StockInfo struct {
	StockID     string `json:"stock_id" gorm:"column:stock_id;type:char(30);"`
	StockIDNum  string `json:"stock_id_num" gorm:"column:stock_id_num;type:char(30);"`
	DisplayName string `json:"display_name" gorm:"column:display_name;type:char(100);"`
	Name        string `json:"name" gorm:"column:name;type:char(100);"`
}

func (s *StockInfo) TableName() string {
	return "stock_info"
}

func TestBaseRepo_Find(t *testing.T) {
	repo := database.BaseRepo{
		Db:    db,
		Model: &StockInfo{},
	}
	list := []*StockInfo{}
	cnt, err := repo.Find(context.Background(), &list, utils.MI{}, nil)
	if err != nil {
		t.Errorf("Find() error = %v", err)
		return
	}
	if cnt != 0 {
		t.Errorf("Find() cnt = %v, want 0", cnt)
		return
	}
	for _, item := range list {
		fmt.Printf("item = %#v\n", item)
	}
}

func TestBaseRepo_FindOne(t *testing.T) {
	repo := database.BaseRepo{
		Db:    db,
		Model: &StockInfo{},
	}
	item := &StockInfo{}
	err := repo.FindOne(context.Background(), item, utils.MI{})
	if err != nil {
		t.Errorf("FindOne() error = %v", err)
		return
	}
	fmt.Printf("item = %#v\n", item)
}
