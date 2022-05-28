package executor

// 响应码
type ResponseCode = int64

const (
	Success ResponseCode = 200
	Failure ResponseCode = 500
)

var (
	DefaultExecutorPort = "9999"
	DefaultRegistryKey  = "ticktock"
)
