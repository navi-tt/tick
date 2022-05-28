package executor

import (
	"context"
	"fmt"
	"runtime/debug"
)

// JobFunc 任务执行函数
type JobFunc func(ctx context.Context, param *RunRequest) string

// Job 任务
type Job struct {
	Id        int64
	Name      string
	Ctx       context.Context
	Param     *RunRequest
	fn        JobFunc
	Cancel    context.CancelFunc
	StartTime int64
	EndTime   int64
	//日志
	log Logger
}

// Run 运行任务
func (t *Job) Run(callback func(code int64, msg string)) {
	defer func(cancel func()) {
		if err := recover(); err != nil {
			t.log.Info(t.Info()+" panic: %v", err)
			debug.PrintStack() //堆栈跟踪
			callback(Failure, fmt.Sprintf("job panic:%v", err))
			cancel()
		}
	}(t.Cancel)
	msg := t.fn(t.Ctx, t.Param)
	callback(Success, msg)
	return
}

// Info 任务信息
func (t *Job) Info() string {
	return fmt.Sprintf("任务ID[%d]任务名称[%s]参数:%s", t.Id, t.Name, t.Param.ExecutorParams)
}
