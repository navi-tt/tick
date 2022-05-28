package executor

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Executor 执行器
type Executor interface {
	// Init 初始化
	Init(...Option)
	// LogHandler 日志查询
	LogHandler(handler LogHandler)
	// RegJob 注册任务
	RegJob(pattern string, job JobFunc)
	// RunJob 运行任务
	RunJob(w http.ResponseWriter, r *http.Request)
	// KillJob 杀死任务
	KillJob(w http.ResponseWriter, r *http.Request)
	// JobLog 任务日志
	JobLog(w http.ResponseWriter, r *http.Request)
	// Beat 心跳检测
	Beat(w http.ResponseWriter, r *http.Request)
	// IdleBeat 忙碌检测
	IdleBeat(w http.ResponseWriter, r *http.Request)
	// Run 运行服务
	Run() error
	// Stop 停止服务
	Stop()
}

// NewExecutor 创建执行器
func NewExecutor(opts ...Option) Executor {
	return newExecutor(opts...)
}

func newExecutor(opts ...Option) *executor {
	options := newOptions(opts...)
	return &executor{
		opts: options,
	}
}

type executor struct {
	opts    Options
	address string
	regJobs *Manager // 注册任务列表
	runJobs *Manager // 正在执行任务列表
	mu      sync.RWMutex
	log     Logger

	logHandler LogHandler // 日志查询handler
}

func (e *executor) Init(opts ...Option) {
	for _, o := range opts {
		o(&e.opts)
	}
	e.log = e.opts.logger
	e.regJobs = &Manager{
		data: make(map[string]*Job),
	}
	e.runJobs = &Manager{
		data: make(map[string]*Job),
	}
	e.address = e.opts.ExecutorIp + ":" + e.opts.ExecutorPort
	go e.registry()
}

// LogHandler 日志handler
func (e *executor) LogHandler(handler LogHandler) {
	e.logHandler = handler
}

func (e *executor) Run() (err error) {
	// 创建路由器
	mux := http.NewServeMux()
	// 设置路由规则
	mux.HandleFunc("/run", e.runJob)
	mux.HandleFunc("/kill", e.killJob)
	mux.HandleFunc("/log", e.jobLog)
	mux.HandleFunc("/beat", e.beat)
	mux.HandleFunc("/idleBeat", e.idleBeat)
	// 创建服务器
	server := &http.Server{
		Addr:         e.address,
		WriteTimeout: time.Second * 3,
		Handler:      mux,
	}
	// 监听端口并提供服务
	e.log.Info("Starting server at " + e.address)
	go server.ListenAndServe()
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	e.registryRemove()
	return nil
}

func (e *executor) Stop() {
	e.registryRemove()
}

// RegJob 注册任务
func (e *executor) RegJob(pattern string, job JobFunc) {
	var t = &Job{}
	t.fn = job
	e.regJobs.SetJob(pattern, t)
	return
}

//运行一个任务
func (e *executor) runJob(w http.ResponseWriter, r *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(r.Body)
	param := &RunRequest{}
	err := json.Unmarshal(req, &param)
	if err != nil {
		_, _ = w.Write(returnCall(param, Failure, "params err"))
		e.log.Error("参数解析错误:" + string(req))
		return
	}
	e.log.Info("任务参数:%v", param)
	if !e.regJobs.Exists(param.ExecutorHandler) {
		_, _ = w.Write(returnCall(param, Failure, "Job not registered"))
		e.log.Error("任务[" + Int64ToStr(param.JobID) + "]没有注册:" + param.ExecutorHandler)
		return
	}

	//阻塞策略处理
	if e.runJobs.Exists(Int64ToStr(param.JobID)) {
		if param.ExecutorBlockStrategy == CoverEarly { //覆盖之前调度
			oldJob := e.runJobs.GetJob(Int64ToStr(param.JobID))
			if oldJob != nil {
				oldJob.Cancel()
				e.runJobs.DelJob(Int64ToStr(oldJob.Id))
			}
		} else { //单机串行,丢弃后续调度 都进行阻塞
			_, _ = w.Write(returnCall(param, Failure, "There are jobs running"))
			e.log.Error("任务[" + Int64ToStr(param.JobID) + "]已经在运行了:" + param.ExecutorHandler)
			return
		}
	}

	cxt := r.Context()
	job := e.regJobs.GetJob(param.ExecutorHandler)
	if param.ExecutorTimeout > 0 {
		job.Ctx, job.Cancel = context.WithTimeout(cxt, time.Duration(param.ExecutorTimeout)*time.Second)
	} else {
		job.Ctx, job.Cancel = context.WithCancel(cxt)
	}
	job.Id = param.JobID
	job.Name = param.ExecutorHandler
	job.Param = param
	job.log = e.log

	e.runJobs.SetJob(Int64ToStr(job.Id), job)
	go job.Run(func(code int64, msg string) {
		e.callback(job, code, msg)
	})
	e.log.Info("任务[" + Int64ToStr(param.JobID) + "]开始执行:" + param.ExecutorHandler)
	_, _ = w.Write(returnGeneral())
}

//删除一个任务
func (e *executor) killJob(w http.ResponseWriter, r *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(r.Body)
	param := &KillRequest{}
	_ = json.Unmarshal(req, &param)
	if !e.runJobs.Exists(Int64ToStr(param.JobID)) {
		_, _ = w.Write(returnKill(param, Failure))
		e.log.Error("任务[" + Int64ToStr(param.JobID) + "]没有运行")
		return
	}
	job := e.runJobs.GetJob(Int64ToStr(param.JobID))
	job.Cancel()
	e.runJobs.DelJob(Int64ToStr(param.JobID))
	_, _ = w.Write(returnGeneral())
}

//任务日志
func (e *executor) jobLog(w http.ResponseWriter, r *http.Request) {
	var res *LogResponse
	data, err := ioutil.ReadAll(r.Body)
	req := &LogRequest{}
	if err != nil {
		e.log.Error("日志请求失败:" + err.Error())
		reqErrLogHandler(w, req, err)
		return
	}
	err = json.Unmarshal(data, &req)
	if err != nil {
		e.log.Error("日志请求解析失败:" + err.Error())
		reqErrLogHandler(w, req, err)
		return
	}
	e.log.Info("日志请求参数:%+v", req)
	if e.logHandler != nil {
		res = e.logHandler(req)
	} else {
		res = defaultLogHandler(req)
	}
	str, _ := json.Marshal(res)
	_, _ = w.Write(str)
}

// 心跳检测
func (e *executor) beat(w http.ResponseWriter, r *http.Request) {
	e.log.Info("心跳检测")
	_, _ = w.Write(returnGeneral())
}

// 忙碌检测
func (e *executor) idleBeat(w http.ResponseWriter, r *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(r.Body)
	param := &IdleBeatRequest{}
	err := json.Unmarshal(req, &param)
	if err != nil {
		_, _ = w.Write(returnIdleBeat(Failure))
		e.log.Error("参数解析错误:" + string(req))
		return
	}
	if e.runJobs.Exists(Int64ToStr(param.JobID)) {
		_, _ = w.Write(returnIdleBeat(Failure))
		e.log.Error("idleBeat任务[" + Int64ToStr(param.JobID) + "]正在运行")
		return
	}
	e.log.Info("忙碌检测任务参数:%v", param)
	_, _ = w.Write(returnGeneral())
}

//注册执行器到调度中心
func (e *executor) registry() {

	t := time.NewTimer(time.Second * 0) //初始立即执行
	defer t.Stop()
	req := &Registry{
		RegistryGroup: "EXECUTOR",
		RegistryKey:   e.opts.RegistryKey,
		RegistryValue: "http://" + e.address,
	}
	param, err := json.Marshal(req)
	if err != nil {
		log.Fatal("执行器注册信息解析失败:" + err.Error())
	}
	for {
		<-t.C
		t.Reset(time.Second * time.Duration(20)) //20秒心跳防止过期
		func() {
			result, err := e.post("/api/registry", string(param))
			if err != nil {
				e.log.Error("执行器注册失败1:" + err.Error())
				return
			}
			defer result.Body.Close()
			body, err := ioutil.ReadAll(result.Body)
			if err != nil {
				e.log.Error("执行器注册失败2:" + err.Error())
				return
			}
			res := &Response{}
			_ = json.Unmarshal(body, &res)
			if res.Code != Success {
				e.log.Error("执行器注册失败3:" + string(body))
				return
			}
			e.log.Info("执行器注册成功:" + string(body))
		}()

	}
}

//执行器注册摘除
func (e *executor) registryRemove() {
	t := time.NewTimer(time.Second * 0) //初始立即执行
	defer t.Stop()
	req := &Registry{
		RegistryGroup: "EXECUTOR",
		RegistryKey:   e.opts.RegistryKey,
		RegistryValue: "http://" + e.address,
	}
	param, err := json.Marshal(req)
	if err != nil {
		e.log.Error("执行器摘除失败:" + err.Error())
		return
	}
	res, err := e.post("/api/registryRemove", string(param))
	if err != nil {
		e.log.Error("执行器摘除失败:" + err.Error())
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	e.log.Info("执行器摘除成功:" + string(body))
	_ = res.Body.Close()
}

//回调任务列表
func (e *executor) callback(job *Job, code int64, msg string) {
	e.runJobs.DelJob(Int64ToStr(job.Id))
	res, err := e.post("/api/callback", string(returnCall(job.Param, code, msg)))
	if err != nil {
		e.log.Error("callback err : ", err.Error())
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		e.log.Error("callback ReadAll err : ", err.Error())
		return
	}
	e.log.Info("任务回调成功:" + string(body))
}

//post
func (e *executor) post(action, body string) (resp *http.Response, err error) {
	request, err := http.NewRequest("POST", e.opts.ServerAddr+action, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	request.Header.Set("XXL-JOB-ACCESS-TOKEN", e.opts.AccessToken)
	client := http.Client{
		Timeout: e.opts.Timeout,
	}
	return client.Do(request)
}

// RunJob 运行任务
func (e *executor) RunJob(w http.ResponseWriter, r *http.Request) {
	e.runJob(w, r)
}

// KillJob 删除任务
func (e *executor) KillJob(w http.ResponseWriter, r *http.Request) {
	e.killJob(w, r)
}

// JobLog 任务日志
func (e *executor) JobLog(w http.ResponseWriter, r *http.Request) {
	e.jobLog(w, r)
}

// Beat 心跳检测
func (e *executor) Beat(w http.ResponseWriter, r *http.Request) {
	e.beat(w, r)
}

// IdleBeat 忙碌检测
func (e *executor) IdleBeat(w http.ResponseWriter, r *http.Request) {
	e.idleBeat(w, r)
}
