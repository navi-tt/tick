package executor

import (
	"encoding/json"
	"net/http"
)

/**
用来日志查询，显示到xxl-job-admin后台
*/

type LogHandler func(req *LogRequest) *LogResponse

//默认返回
func defaultLogHandler(req *LogRequest) *LogResponse {
	return &LogResponse{
		Code: Success,
		Msg:  "",
		Content: LogContent{
			FromLineNum: req.FromLineNum,
			ToLineNum:   2,
			LogContent:  "这是日志默认返回，说明没有设置LogHandler",
			IsEnd:       true,
		}}
}

//请求错误
func reqErrLogHandler(w http.ResponseWriter, req *LogRequest, err error) {
	res := &LogResponse{Code: Failure, Msg: err.Error(), Content: LogContent{
		FromLineNum: req.FromLineNum,
		ToLineNum:   0,
		LogContent:  err.Error(),
		IsEnd:       true,
	}}
	str, _ := json.Marshal(res)
	_, _ = w.Write(str)
}
