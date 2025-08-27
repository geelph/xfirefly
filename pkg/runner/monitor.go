package runner

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/donnie4w/go-logger/logger"
)

// MemoryStats 内存统计信息
type MemoryStats struct {
	HeapAlloc     uint64    // 堆已分配内存 (字节)
	HeapSys       uint64    // 堆系统内存 (字节)
	NumGC         uint32    // GC次数
	GCCPUFraction float64   // GC占用CPU时间比例
	LastGCTime    time.Time // 上次GC时间
	MemoryUsage   float64   // 内存使用率 (%)
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	enabled              atomic.Bool
	lastGCTime           int64
	highMemThreshold     uint64 // 高内存使用阈值
	criticalMemThreshold uint64 // 临界内存使用阈值
}

// 声明性能监视器变量
var globalMonitor *PerformanceMonitor

// 初始化全局监控器
func init() {
	// 全局性能监控器变量赋值
	globalMonitor = &PerformanceMonitor{
		highMemThreshold:     2 * 1024 * 1024 * 1024, // 2GB
		criticalMemThreshold: 4 * 1024 * 1024 * 1024, // 4GB
	}
}

// StartMemoryMonitor 启动内存监控
func StartMemoryMonitor() {
	// 确保性能监控器已经启动
	if !globalMonitor.enabled.CompareAndSwap(false, true) {
		return // 已经启动
	}

	// 开启一个异步监控线程
	go globalMonitor.monitorLoop()
	// 打印日志信息
	logger.Info("内存监控已启动")
}

// StopMemoryMonitor 停止内存监控
func StopMemoryMonitor() {
	// 监控线程状态设置为false
	globalMonitor.enabled.Store(false)
	logger.Info("内存监控已停止")
}

// monitorLoop
//
//	@Description: 定义 PerformanceMonitor 的监控循环监控
//	@receiver pm PerformanceMonitor类型变量
func (pm *PerformanceMonitor) monitorLoop() {
	// 定时触发的计时器
	ticker := time.NewTicker(30 * time.Second) // 30秒检查一次
	// 确保计时器被手动关闭
	defer ticker.Stop()

	// 并发安全避免使用互斥锁
	for pm.enabled.Load() {
		// 等待ticker触发
		select {
		case <-ticker.C:
			// 触发后检查内存使用情况
			pm.checkMemoryUsage()
		}
	}
}

// checkMemoryUsage 检查内存使用情况
//
//	@Description: 定义 PerformanceMonitor 的检查内存使用情况方法
//	@receiver pm PerformanceMonitor 变量
func (pm *PerformanceMonitor) checkMemoryUsage() {
	// 声明内存状态变量
	var memStats runtime.MemStats
	// 读取内存状态
	runtime.ReadMemStats(&memStats)

	// 只读取需要的内存状态信息
	// 其实可以调用 GetMemoryStats() 函数来获取不需要再重复写一遍代码
	stats := MemoryStats{
		HeapAlloc:     memStats.HeapAlloc,
		HeapSys:       memStats.HeapSys,
		NumGC:         memStats.NumGC,
		GCCPUFraction: memStats.GCCPUFraction,
		LastGCTime:    time.Unix(0, int64(memStats.LastGC)),
		MemoryUsage:   float64(memStats.HeapAlloc) / float64(memStats.HeapSys) * 100,
	}

	// 记录内存使用情况
	logger.Debug(fmt.Sprintf("内存使用: %.2f MB (%.1f%%), GC次数: %d",
		float64(stats.HeapAlloc)/1024/1024, stats.MemoryUsage, stats.NumGC))

	// 根据内存使用情况采取措施
	pm.handleMemoryPressure(&stats)
}

// handleMemoryPressure 处理内存压力
//
//	@Description: 定义处理内存压力的方法，内存使用超过阈值、85%、距离上次GC超过2分钟
//	@receiver pm PerformanceMonitor 变量
//	@param stats 内存状态
func (pm *PerformanceMonitor) handleMemoryPressure(stats *MemoryStats) {
	// 检查是否需要强制GC
	shouldForceGC := false

	// 条件1: 内存使用超过高阈值
	if stats.HeapAlloc > pm.highMemThreshold {
		shouldForceGC = true
		logger.Debug("内存使用超过高阈值，触发GC")
	}

	// 条件2: 内存使用率超过85%
	if stats.MemoryUsage > 85.0 {
		shouldForceGC = true
		logger.Debug("内存使用率过高，触发GC")
	}

	// 条件3: 距离上次GC时间超过2分钟
	if time.Since(stats.LastGCTime) > 2*time.Minute {
		shouldForceGC = true
		logger.Debug("距离上次GC时间过长，触发GC")
	}

	// 强制GC
	if shouldForceGC {
		// 可调用编写的强制GC函数
		// 强制GC
		runtime.GC()

		// 如果内存使用仍然很高，释放系统内存
		if stats.HeapAlloc > pm.criticalMemThreshold {
			logger.Debug("内存使用达到临界值，释放系统内存")
			// 主动释放未使用的内存
			debug.FreeOSMemory()
		}
	}
}

// GetMemoryStats 获取当前内存统计信息
func GetMemoryStats() MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return MemoryStats{
		HeapAlloc:     memStats.HeapAlloc,
		HeapSys:       memStats.HeapSys,
		NumGC:         memStats.NumGC,
		GCCPUFraction: memStats.GCCPUFraction,
		LastGCTime:    time.Unix(0, int64(memStats.LastGC)),
		MemoryUsage:   float64(memStats.HeapAlloc) / float64(memStats.HeapSys) * 100,
	}
}

// ForceGC 强制执行垃圾回收
func ForceGC() {
	before := GetMemoryStats()
	runtime.GC()
	after := GetMemoryStats()

	logger.Debug(fmt.Sprintf("强制GC执行完成，内存释放: %.2f MB",
		float64(before.HeapAlloc-after.HeapAlloc)/1024/1024))
}

// SetMemoryThresholds 设置内存阈值
//
// SetMemoryThresholds 设置内存阈值
//
//	@Description: 设置内存阈值，外部调用
//	@param highThreshold 高阈值，单位B
//	@param criticalThreshold 临界值，单位B
func SetMemoryThresholds(highThreshold, criticalThreshold uint64) {
	globalMonitor.highMemThreshold = highThreshold
	globalMonitor.criticalMemThreshold = criticalThreshold
	logger.Info(fmt.Sprintf("内存阈值已更新: 高阈值=%.2f MB, 临界阈值=%.2f MB",
		float64(highThreshold)/1024/1024, float64(criticalThreshold)/1024/1024))
}
