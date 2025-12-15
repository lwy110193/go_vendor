package crontab

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// TaskConfig 任务配置
type TaskConfig struct {
	Name               string `yaml:"name"`                // 任务名称
	Spec               string `yaml:"spec"`                // 任务表达式
	ExecuteImmediately bool   `yaml:"execute_immediately"` // 是否启动时立即执行
	Enabled            bool   `yaml:"enabled"`             // 启用状态
}

type TaskLogInterface interface {
	WriteLog(ctx context.Context, msg string, keysAndValues ...interface{})
	FatalLog(ctx context.Context, msg string, keysAndValues ...interface{})
}

// Task 接口表示一个可运行的任务。
type Task interface {
	// Name 任务名称
	GetName() string

	// Desc 任务描述
	GetDesc() string

	// Run 方法运行任务的逻辑。
	Run()

	Log() TaskLogInterface

	// SetParam 设置参数
	SetParam(param interface{}) error
}

// Register 已注册任务
var list []Task

// Register 注册任务初始化函数
func Register(t Task) {
	list = append(list, t)
}

// GetRegisteredList 获取已注册任务
func GetRegisteredList() []Task {
	return list
}

// Run 初始化所有 task 并启动任务
func Run(tasks []*TaskConfig) {
	c := cron.New(cron.WithSeconds())

	conf := getTaskConfig(tasks)
	for _, taskItem := range list {
		name := taskItem.GetName()
		cfg, exist := conf[name]
		if !exist {
			continue
		}
		if !cfg.Enabled {
			continue
		}
		if cfg.ExecuteImmediately {
			taskItem.Log().WriteLog(context.Background(), fmt.Sprintf("%sexecute immediately", time.Now().Format("2006-01-02 15:04:05")))
			go taskItem.Run()
		}
		_, err := c.AddFunc(cfg.Spec, taskItem.Run)
		if err != nil {
			taskItem.Log().FatalLog(context.Background(), fmt.Sprintf("[timer Add Task: %s, conf: %+v, err: %v]", taskItem.GetDesc(), cfg, err))
			panic(0)
		}
		taskItem.Log().WriteLog(context.Background(), fmt.Sprintf("[timer Add Task: %s, conf: %+v]", taskItem.GetDesc(), cfg))
	}
	c.Start()
}

// getTaskConfig 从任务配置列表中构建任务配置映射
func getTaskConfig(tasks []*TaskConfig) map[string]*TaskConfig {
	m := make(map[string]*TaskConfig)
	for _, t := range tasks {
		m[t.Name] = t
	}

	return m
}
