package crontab_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/lwy110193/go_vendor/crontab"
)

type TaskLogger struct{}

func (l *TaskLogger) WriteLog(ctx context.Context, format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *TaskLogger) FatalLog(ctx context.Context, format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

type TestTask struct {
	Name string
	Desc string
}

func (t *TestTask) GetDesc() string {
	return t.Desc
}

func (t *TestTask) SetParam(param interface{}) error {
	return nil
}

func (t *TestTask) GetName() string {
	return t.Name
}

func (t *TestTask) Log() crontab.TaskLogInterface {
	return &TaskLogger{}
}

func (t *TestTask) Run() {
	t.Log().WriteLog(context.Background(), fmt.Sprintf("%s[%s] runing", t.GetDesc(), t.GetName()))
}

func TestAddTask(t *testing.T) {
	t.Setenv("CRON_LOG_LEVEL", "debug")

	taskConfig := []*crontab.TaskConfig{
		{
			Name:               "test_task",
			Enabled:            true,
			ExecuteImmediately: true,
			Spec:               "*/5 * * * * *",
		},
	}

	taskItem := &TestTask{Name: "test_task", Desc: "test_task_desc"}
	crontab.Register(taskItem)
	defer func() {
		if err := recover(); err != nil {
			taskItem.Log().FatalLog(context.Background(), "crontab run failed: %v", err)
		} else {
			taskItem.Log().WriteLog(context.Background(), "crontab run success")
		}
	}()

	crontab.Run(taskConfig)

	time.Sleep(100 * time.Second)
}
