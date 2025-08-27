package runner

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"xfirefly/pkg/finger"

	"github.com/donnie4w/go-logger/logger"
	"github.com/panjf2000/ants/v2"
)

// Pool 抽象的工作池接口，屏蔽对 ants 的直接依赖
type Pool interface {
	Invoke(i interface{}) error
	Release()
}

// antsPoolWrapper 使用 ants.PoolWithFunc 实现 Pool 接口
type antsPoolWrapper struct {
	inner *ants.PoolWithFunc
}

func (p *antsPoolWrapper) Invoke(i interface{}) error { return p.inner.Invoke(i) }
func (p *antsPoolWrapper) Release()                   { p.inner.Release() }

// NewWorkPoolWithFunc 创建一个带函数处理器的工作池
// 统一在此集中 ants 相关实现
func NewWorkPoolWithFunc(
	workerCount int,
	handler func(interface{}),
	maxBlockingTasks int,
	expiry time.Duration,
	panicHandler func(interface{}),
) (Pool, error) {
	pool, err := ants.NewPoolWithFunc(
		workerCount,
		handler,
		ants.WithPreAlloc(true),
		ants.WithExpiryDuration(expiry),
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(maxBlockingTasks),
		ants.WithPanicHandler(panicHandler),
	)
	if err != nil {
		return nil, err
	}
	return &antsPoolWrapper{inner: pool}, nil
}

// ===================== 规则池封装 =====================

// GlobalRulePoolStats 全局规则池统计信息
type GlobalRulePoolStats struct {
	TotalTasks     int64 // 成功提交的总任务数
	CompletedTasks int64 // 已完成任务数
	FailedTasks    int64 // 失败任务数
}

var (
	// 规则池实例（对外仅通过函数访问）
	globalRulePool Pool
	// 池统计
	rulePoolStats GlobalRulePoolStats
)

// RuleTask 规则处理任务结构（供调用方构造任务使用）
type RuleTask struct {
	Target     string
	Finger     *finger.Finger
	BaseInfo   *BaseInfo
	Proxy      string
	Timeout    int
	ResultChan chan<- *FingerMatch // 结果通道
	WaitGroup  *sync.WaitGroup     // 等待组
}

// InitGlobalRulePool 初始化全局规则处理池
func InitGlobalRulePool(workerCount int, fingerActive bool) error {
	// 规则池 handler，集中处理单个任务
	handler := func(i interface{}) {
		task, ok := i.(*RuleTask)
		if !ok {
			atomic.AddInt64(&rulePoolStats.FailedTasks, 1)
			logger.Error("无效的规则任务类型")
			return
		}

		processRuleTask(task, fingerActive)

		// 完成计数
		atomic.AddInt64(&rulePoolStats.CompletedTasks, 1)
	}

	pool, err := NewWorkPoolWithFunc(
		workerCount,
		handler,
		workerCount*10,
		2*time.Minute,
		func(i interface{}) {
			atomic.AddInt64(&rulePoolStats.FailedTasks, 1)
			logger.Error(fmt.Sprintf("规则池goroutine异常: %v", i))
		},
	)
	if err != nil {
		return fmt.Errorf("创建全局规则池失败: %v", err)
	}

	globalRulePool = pool
	logger.Info(fmt.Sprintf("全局规则池初始化完成，工作线程数: %d", workerCount))
	return nil
}

// ReleaseRulePool 释放全局规则池
func ReleaseRulePool() {
	if globalRulePool != nil {
		globalRulePool.Release()
		globalRulePool = nil
	}
}

// IsRulePoolInitialized 是否已初始化全局规则池
func IsRulePoolInitialized() bool { return globalRulePool != nil }

// SubmitRuleTask 提交规则任务到全局规则池
func SubmitRuleTask(task *RuleTask) error {
	if globalRulePool == nil {
		return fmt.Errorf("全局规则池未初始化")
	}
	if err := globalRulePool.Invoke(task); err != nil {
		return err
	}
	atomic.AddInt64(&rulePoolStats.TotalTasks, 1)
	return nil
}

// GetRulePoolStats 获取全局规则池统计信息
func GetRulePoolStats() GlobalRulePoolStats {
	return GlobalRulePoolStats{
		TotalTasks:     atomic.LoadInt64(&rulePoolStats.TotalTasks),
		CompletedTasks: atomic.LoadInt64(&rulePoolStats.CompletedTasks),
		FailedTasks:    atomic.LoadInt64(&rulePoolStats.FailedTasks),
	}
}

// GetPoolStats 对外统一命名的统计获取函数（与对外API一致）
func GetPoolStats() GlobalRulePoolStats { return GetRulePoolStats() }

// ResetPoolStats 重置统计计数
func ResetPoolStats() {
	atomic.StoreInt64(&rulePoolStats.TotalTasks, 0)
	atomic.StoreInt64(&rulePoolStats.CompletedTasks, 0)
	atomic.StoreInt64(&rulePoolStats.FailedTasks, 0)
}

// processRuleTask 处理单个规则识别任务
func processRuleTask(task *RuleTask, fingerActive bool) {
	defer func() {
		if task.WaitGroup != nil {
			task.WaitGroup.Done()
		}
	}()

	// 执行指纹识别
	result, err := evaluateFingerprintWithCache(
		task.Finger,
		task.Target,
		task.BaseInfo,
		task.Proxy,
		task.Timeout,
		fingerActive,
	)

	if err != nil {
		atomic.AddInt64(&rulePoolStats.FailedTasks, 1)
		logger.Debug(fmt.Sprintf("规则 %s 执行失败: %v", task.Finger.Id, err))
		return
	}

	// 只有匹配成功的结果才发送到结果通道
	if result != nil && result.Result {
		select {
		case task.ResultChan <- result:
		default:
			logger.Debug(fmt.Sprintf("结果通道已满，丢弃规则 %s 的结果", task.Finger.Id))
		}
	}
}
