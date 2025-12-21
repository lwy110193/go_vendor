package database_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lwy110193/db_define/stock/models"
	"github.com/lwy110193/go_vendor/database"
	"github.com/lwy110193/go_vendor/log"
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

type TeTable struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id;comment:id"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;index;autoCreateTime;not null;comment:创建时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;index;autoCreateTime;not null;comment:更新时间"`
	DeletedAt time.Time `gorm:"column:deleted_at;type:datetime;default:null;comment:删除时间"`
	Field1    string    `gorm:"column:field1;type:char(20);comment:字段1"`
	Field2    string    `gorm:"column:field2;type:varchar(100);comment:字段2"`
}

func (t *TeTable) TableName() string {
	return "te_table"
}

func Test_CreateTableTe(t *testing.T) {
	err := db.AutoMigrate(&TeTable{})
	if err != nil {
		t.Errorf("AutoMigrate() error = %v", err)
		return
	}
}

type StockInfoRepo struct {
	database.BaseRepo
	logger log.LogInterface
}

// NewStockInfoRepo 创建新的股票信息仓库
func NewStockInfoRepo(db *gorm.DB, logger log.LogInterface) *StockInfoRepo {
	return &StockInfoRepo{
		BaseRepo: database.BaseRepo{
			Db:    db,
			Model: &models.StockInfo{},
		},
		logger: logger,
	}
}

type Stock struct {
	StockId     string `json:"stock_id" gorm:"column:stock_id;type:char(30);comment:股票ID"`
	DisplayName string `json:"display_name" gorm:"column:display_name;type:char(100);comment:股票名称"`
}

func TestCreateInBatch(t *testing.T) {
	repo := NewStockInfoRepo(db, nil)

	var resultList []*Stock
	count, err := repo.Find(context.Background(), &resultList, utils.MI{}, nil, "stock_id", "display_name")
	if err != nil {
		t.Errorf("Find() error = %v", err)
		return
	}
	fmt.Printf("count = %v\n", count)
	fmt.Printf("resultList = %#v\n", resultList)

	var findOne models.StockInfo
	if err = repo.FindOne(context.Background(), &findOne, utils.MI{}); err != nil {
		t.Errorf("FindOne() error = %v", err)
		return
	}
	fmt.Printf("findOne = %#v\n", findOne)

	time := time.Now()
	list := []*models.StockInfo{
		{
			StockId:     utils.RandNumCode(10),
			DisplayName: utils.RandNumCode(10),
			StartDate:   time,
			EndDate:     time,
		},
		{
			StockId:     utils.RandNumCode(10),
			DisplayName: utils.RandNumCode(10),
			StartDate:   time,
			EndDate:     time,
		},
	}

	if err = repo.CreateBatch(context.Background(), list, 100); err != nil {
		t.Errorf("CreateBatch() error = %v", err)
		return
	}

	one := &models.StockInfo{
		StockId:     utils.RandNumCode(10),
		DisplayName: utils.RandNumCode(10),
		StartDate:   time,
		EndDate:     time,
	}
	if err = repo.Create(context.Background(), one); err != nil {
		t.Errorf("Create() error = %v", err)
		return
	}

	sql := "select * from stock_info where stock_id = ?"
	var resultList2 []*models.StockInfo
	if err = repo.Raw(context.Background(), &resultList2, sql, "3996601512"); err != nil {
		t.Errorf("Raw() error = %v", err)
		return
	}
	fmt.Printf("resultList2 = %#v\n", resultList2)

	if err = repo.Exec(context.Background(), "update stock_info set display_name = ? where stock_id = ?", "新名称", "3996601512"); err != nil {
		t.Errorf("Exec() error = %v", err)
		return
	}

	// repo.UpdateInBatchForStruct(context.Background(), list, utils.MI{
	// 	"stock_id": one.StockId,
	// }, []string{"display_name"}, []string{"start_date", "end_date"}); err != nil {
	// 	t.Errorf("UpdateInBatchForStruct() error = %v", err)
	// 	return
	// }

	if err = repo.Transaction(context.Background(), func(tx *gorm.DB) error {
		newOne := &models.StockInfo{
			StockId:     utils.RandNumCode(10),
			DisplayName: "new_one",
			StartDate:   time,
			EndDate:     time,
		}
		if err = tx.Create(newOne).Error; err != nil {
			return err
		}
		return nil
		// return errors.New("dont create new_one")
	}); err != nil {
		t.Errorf("Transaction() error = %v", err)
		return
	}
}
