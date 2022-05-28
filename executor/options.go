package executor

import (
	"time"
)

type Options struct {
	ServerAddr   string        `json:"server_addr"`   //调度中心地址
	AccessToken  string        `json:"access_token"`  //请求令牌
	Timeout      time.Duration `json:"timeout"`       //接口超时时间
	ExecutorIp   string        `json:"executor_ip"`   //本地(执行器)IP(可自行获取)
	ExecutorPort string        `json:"executor_port"` //本地(执行器)端口
	RegistryKey  string        `json:"registry_key"`  //执行器名称
	LogDir       string        `json:"log_dir"`       //日志目录

	logger Logger //日志处理
}

func newOptions(opts ...Option) Options {
	opt := Options{
		ExecutorIp:   LocalIP(),
		ExecutorPort: DefaultExecutorPort,
		RegistryKey:  DefaultRegistryKey,
	}

	for _, o := range opts {
		o(&opt)
	}

	if opt.logger == nil {
		opt.logger = &logger{}
	}

	return opt
}

type Option func(o *Options)

// ServerAddr 设置调度中心地址
func ServerAddr(addr string) Option {
	return func(o *Options) {
		o.ServerAddr = addr
	}
}

// AccessToken 请求令牌
func AccessToken(token string) Option {
	return func(o *Options) {
		o.AccessToken = token
	}
}

// LocalExecutorIp 设置执行器IP
func LocalExecutorIp(ip string) Option {
	return func(o *Options) {
		o.ExecutorIp = ip
	}
}

// LocalExecutorPort 设置执行器端口
func LocalExecutorPort(port string) Option {
	return func(o *Options) {
		o.ExecutorPort = port
	}
}

// RegistryKey 设置执行器标识
func RegistryKey(registryKey string) Option {
	return func(o *Options) {
		o.RegistryKey = registryKey
	}
}

// SetLogger 设置日志处理器
func SetLogger(logger Logger) Option {
	return func(o *Options) {
		o.logger = logger
	}
}
