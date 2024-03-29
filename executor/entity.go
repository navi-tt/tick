package executor

//通用响应
type Response struct {
	Code ResponseCode `json:"code"` // 200 表示正常、其他失败
	Msg  interface{}  `json:"msg"`  // 错误提示消息
}

/*****************  上行参数  *********************/

// Registry 注册参数
type Registry struct {
	RegistryGroup string `json:"registry_group"`
	RegistryKey   string `json:"registry_key"`
	RegistryValue string `json:"registry_value"`
}

//执行器执行完任务后，回调任务结果时使用
type call []*callElement

type callElement struct {
	LogID         int64          `json:"log_id"`
	LogDateTim    int64          `json:"log_date_tim"`
	ExecuteResult *ExecuteResult `json:"execute_result"`
	HandleCode    int            `json:"handle_code"` //200表示正常,500表示失败
	HandleMsg     string         `json:"handle_msg"`
}

// ExecuteResult 任务执行结果 200 表示任务执行正常，500表示失败
type ExecuteResult struct {
	Response
}

/*****************  下行参数  *********************/

//阻塞处理策略
const (
	SerialExecution = "SERIAL_EXECUTION" // 单机串行
	DiscardLater    = "DISCARD_LATER"    // 丢弃后续调度
	CoverEarly      = "COVER_EARLY"      // 覆盖之前调度
)

// RunRequest 触发任务请求参数
type RunRequest struct {
	JobID                 int64  `json:"job_id"`                  // 任务ID
	ExecutorHandler       string `json:"executor_handler"`        // 任务标识
	ExecutorParams        string `json:"executor_params"`         // 任务参数
	ExecutorBlockStrategy string `json:"executor_block_strategy"` // 任务阻塞策略
	ExecutorTimeout       int64  `json:"executor_timeout"`        // 任务超时时间，单位秒，大于零时生效
	LogID                 int64  `json:"log_id"`                  // 本次调度日志ID
	LogDateTime           int64  `json:"log_date_time"`           // 本次调度日志时间
	GlueType              string `json:"glue_type"`               // 任务模式，可选值参考 com.xxl.job.core.glue.GlueTypeEnum
	GlueSource            string `json:"glue_source"`             // GLUE脚本代码
	GlueUpdateTime        int64  `json:"glue_update_time"`        // GLUE脚本更新时间，用于判定脚本是否变更以及是否需要刷新
	BroadcastIndex        int64  `json:"broadcast_index"`         // 分片参数：当前分片
	BroadcastTotal        int64  `json:"broadcast_total"`         // 分片参数：总分片
}

// 终止任务请求参数
type KillRequest struct {
	JobID int64 `json:"job_id"` // 任务ID
}

//忙碌检测请求参数
type IdleBeatRequest struct {
	JobID int64 `json:"job_id"` // 任务ID
}

// LogRequest 日志请求
type LogRequest struct {
	LogDateTime int64 `json:"log_date_time"` // 本次调度日志时间
	LogID       int64 `json:"log_id"`        // 本次调度日志ID
	FromLineNum int   `json:"from_line_num"` // 日志开始行号，滚动加载日志
}

// LogResponse 日志响应
type LogResponse struct {
	Code    ResponseCode `json:"code"`    // 200 表示正常、其他失败
	Msg     string       `json:"msg"`     // 错误提示消息
	Content LogContent   `json:"content"` // 日志响应内容
}

// LogContent 日志响应内容
type LogContent struct {
	FromLineNum int    `json:"from_line_num"` // 本次请求，日志开始行数
	ToLineNum   int    `json:"to_line_num"`   // 本次请求，日志结束行号
	LogContent  string `json:"log_content"`   // 本次请求日志内容
	IsEnd       bool   `json:"is_end"`        // 日志是否全部加载完
}
