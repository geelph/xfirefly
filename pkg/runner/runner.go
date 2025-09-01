package runner

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"xfirefly/pkg/output"
	"xfirefly/pkg/types"

	"github.com/donnie4w/go-logger/logger"
)

// 全局配置常量 - 导出供外部使用
const (
	DefaultURLWorkers  = 5    // URL处理池默认大小
	DefaultRuleWorkers = 200  // 规则处理池默认大小
	MaxRuleWorkers     = 5000 // 最大规则工作线程
	MinRuleWorkers     = 200  // 最小规则工作线程
)

// Runner 指纹识别运行器
type Runner struct {
	Config    *ScanConfig              // 配置参数
	Results   map[string]*TargetResult // 扫描结果
	mutex     sync.RWMutex             // 读写锁保护Results
	isRunning atomic.Bool              // 运行状态标志
}

// NewRunner 创建一个新的扫描运行器
func NewRunner(options *types.CmdOptionsType) *Runner {
	// 设置URL并发参数，通过参数获取线程数，参数小于0时使用程序默认
	urlWorkerCount := options.Threads
	if urlWorkerCount <= 0 {
		urlWorkerCount = DefaultURLWorkers
	}

	// 设置规则并发参数，默认为500，并进行钳制
	var ruleWorkerCount int
	// 通过参数获取规则线程数
	if options.RuleThreads > 0 {
		ruleWorkerCount = options.RuleThreads
	} else {
		// 使用默认
		ruleWorkerCount = DefaultRuleWorkers
	}
	// 限制最小规则线程
	if ruleWorkerCount < MinRuleWorkers {
		ruleWorkerCount = MinRuleWorkers
	}
	// 限制最大规则线程
	if ruleWorkerCount > MaxRuleWorkers {
		ruleWorkerCount = MaxRuleWorkers
	}

	// 确定输出格式
	// 通过传入的参数
	outputFormat := output.GetOutputFormat(options.JSONOutput, options.Output)

	// 创建配置
	config := &ScanConfig{
		Proxy:             options.Proxy,
		Timeout:           options.Timeout,
		URLWorkerCount:    urlWorkerCount,
		FingerWorkerCount: ruleWorkerCount,
		OutputFormat:      outputFormat,
		OutputFile:        options.Output,
		SockOutputFile:    options.SockOutput,
	}

	// 创建Runner实例
	runner := &Runner{
		Config:  config, // 扫描配置
		Results: make(map[string]*TargetResult),
		mutex:   sync.RWMutex{},
	}

	return runner
}

// Run 执行扫描
func (r *Runner) Run(options *types.CmdOptionsType) error {

	// 检查扫描器是否已经运行
	if !r.isRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("扫描器已在运行中")
	}
	// 确保扫描器停止
	defer r.isRunning.Store(false)

	// 处理目标URL列表
	targets, err := getTargets(options)
	if err == nil {
		// 检测目标有效数
		if len(targets) == 0 {
			return fmt.Errorf("未找到有效的目标URL")
		}
	} else {
		return err
	}

	// 打印扫描目标数
	logger.Info(fmt.Sprintf("准备扫描 %d 个目标", len(targets)))

	// 初始化输出文件
	if r.Config.OutputFile != "" {
		if err := output.InitOutput(r.Config.OutputFile, r.Config.OutputFormat); err != nil {
			return fmt.Errorf("初始化输出文件失败: %v", err)
		}
		logger.Info(fmt.Sprintf("日志输出文件：%s", r.Config.OutputFile))
		defer func() {
			_ = output.Close()
		}()
	}

	// 初始化socket文件输出
	if r.Config.SockOutputFile != "" {
		if err := output.InitSockOutput(r.Config.SockOutputFile); err != nil {
			return fmt.Errorf("初始化socket输出文件失败: %v", err)
		}
		logger.Info(fmt.Sprintf("Socket输出文件：%s", r.Config.SockOutputFile))
	}

	// 加载指纹规则
	if err := LoadFingerprints(options.FingerOptions); err != nil {
		return fmt.Errorf("加载指纹规则出错: %v", err)
	}
	logger.Info(fmt.Sprintf("加载指纹数量：%v个", len(AllFinger)))

	fingerActive := false
	// 是否做主动指纹识别
	if options.Active {
		fingerActive = true
	}

	// 初始化全局规则池
	if !IsRulePoolInitialized() {
		if err := InitGlobalRulePool(r.Config.FingerWorkerCount, fingerActive); err != nil {
			return err
		}
	}

	// 在函数返回时释放全局池资源
	defer ReleaseRulePool()

	logger.Info(fmt.Sprintf("开始扫描 %d 个目标，使用 %d 个URL并发线程, %d 个规则并发线程...",
		len(targets), r.Config.URLWorkerCount, r.Config.FingerWorkerCount))

	// 执行扫描
	if err := r.runScan(targets, options); err != nil {
		return err
	}

	// 清除所有缓存
	ClearAllCache()

	// 打印统计信息
	r.mutex.RLock()
	printSummary(targets, r.Results)
	r.mutex.RUnlock()

	return nil
}

// ScanTarget 扫描单个目标URL
func (r *Runner) ScanTarget(target string) (*TargetResult, error) {
	if !r.isRunning.Load() {
		return nil, fmt.Errorf("扫描器未运行")
	}

	// 处理单个URL
	result, err := ProcessURL(target, r.Config.Proxy, r.Config.Timeout, r.Config.FingerWorkerCount)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// runScan 执行扫描过程
func (r *Runner) runScan(targets []string, options *types.CmdOptionsType) error {
	// 使用较小缓冲通道收集结果，避免为大规模目标一次性分配巨大缓冲区
	resultChan := make(chan struct {
		target string
		result *TargetResult
	}, func() int {
		caps := r.Config.URLWorkerCount * 2
		if caps < 1 {
			caps = 1
		}
		if caps > len(targets) {
			return len(targets)
		}
		return caps
	}())

	// 创建进度条
	bar := output.CreateProgressBar(len(targets))

	// 创建上下文用于控制goroutine
	doneChan := make(chan struct{}, func() int {
		caps := r.Config.URLWorkerCount * 2
		if caps < 1 {
			caps = 1
		}
		if caps > len(targets) {
			return len(targets)
		}
		return caps
	}())
	stopRefreshChan := make(chan struct{})

	// 添加定时刷新进度条的功能
	refreshTicker := time.NewTicker(500 * time.Millisecond)
	go func() {
		defer refreshTicker.Stop()
		for {
			select {
			case <-refreshTicker.C:
				if err := bar.RenderBlank(); err != nil {
					logger.Debug(fmt.Sprintf("刷新进度条出错: %v", err))
				}
			case <-stopRefreshChan:
				return
			}
		}
	}()

	// 启动进度条更新协程
	//startTime := time.Now()
	go func() {
		for range doneChan {
			if err := bar.Add(1); err != nil {
				logger.Debug(fmt.Sprintf("更新进度条出错: %v", err))
			}
		}
	}()

	// 收集结果的协程
	go func() {
		for data := range resultChan {
			r.mutex.Lock()
			r.Results[data.target] = data.result
			r.mutex.Unlock()
		}
	}()

	// 存储输出的结果 - 线程安全的结果输出
	saveResult := func(msg string) {
		fmt.Print("\033[2K\r")
		fmt.Println(msg)
		if err := bar.RenderBlank(); err != nil {
			logger.Debug(fmt.Sprintf("重新显示进度条出错: %v", err))
		}
	}

	// 定义URL处理任务结构体
	type urlTask struct {
		target string
	}

	var urlWg sync.WaitGroup

	// 创建URL处理工作池（通过统一封装）
	pool, err := NewWorkPoolWithFunc(
		r.Config.URLWorkerCount,
		func(i interface{}) {
			defer urlWg.Done()
			task, ok := i.(urlTask)
			if !ok {
				logger.Error("无效的URL任务类型")
				return
			}

			target := task.target

			// 处理单个URL
			targetResult, err := ProcessURL(target, options.Proxy, options.Timeout, r.Config.FingerWorkerCount)
			if err != nil {
				logger.Error(fmt.Sprintf("处理目标 %s 失败: %v", target, err))
				targetResult = &TargetResult{
					URL:     target,
					Matches: make([]*FingerMatch, 0),
				}
			}

			// 将结果写入文件并显示结果
			handleMatchResults(targetResult, options, saveResult, r.Config.OutputFormat)

			// 结果已输出，释放大对象以降低常驻内存
			for _, m := range targetResult.Matches {
				m.Request = nil
				m.Response = nil
			}
			targetResult.LastRequest = nil
			targetResult.LastResponse = nil

			// 通过通道发送结果
			select {
			case resultChan <- struct {
				target string
				result *TargetResult
			}{target, targetResult}:
			default:
				logger.Debug("结果通道已满，丢弃结果")
			}

			// 通知完成一个任务
			select {
			case doneChan <- struct{}{}:
			default:
				logger.Debug("完成通道已满")
			}
		},
		r.Config.URLWorkerCount*5,
		3*time.Minute,
		func(i interface{}) { logger.Error(fmt.Sprintf("URL池goroutine异常: %v", i)) },
	)

	if err != nil {
		return fmt.Errorf("创建URL处理池失败: %v", err)
	}
	defer pool.Release()

	// 提交所有目标到线程池
	for _, target := range targets {
		urlWg.Add(1)
		if err := pool.Invoke(urlTask{target: target}); err != nil {
			urlWg.Done()
			logger.Error(fmt.Sprintf("提交目标 %s 到线程池失败: %v", target, err))
		}
	}

	// 等待当前批次完成
	urlWg.Wait()

	// 等待所有URL处理完成
	close(resultChan)
	close(doneChan)

	// 停止刷新进度条
	close(stopRefreshChan)

	// 确保最终完成100%进度
	if err := bar.Finish(); err != nil {
		logger.Debug(fmt.Sprintf("完成进度条出错: %v", err))
	}

	// 显示扫描耗时信息
	//elapsedTime := time.Since(startTime)
	//itemsPerSecond := float64(len(targets)) / elapsedTime.Seconds()

	//maxProgress := fmt.Sprintf("指纹识别 100%% [==================================================] (%d/%d, %.2f it/s)",
	//	len(targets), len(targets), itemsPerSecond)
	//fmt.Println(maxProgress)

	// 打印池统计信息
	stats := GetRulePoolStats()
	logger.Info(fmt.Sprintf("规则池统计 - 总任务: %d, 已完成: %d, 失败: %d",
		stats.TotalTasks, stats.CompletedTasks, stats.FailedTasks))

	return nil
}
