package runner

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"xfirefly/pkg/finger"
	"xfirefly/pkg/utils/common"
	"xfirefly/pkg/utils/proto"

	"github.com/donnie4w/go-logger/logger"
)

// CacheRequest 存储请求和响应的缓存条目
type CacheRequest struct {
	Request   *proto.Request  `json:"request"`
	Response  *proto.Response `json:"response"`
	Timestamp int64           `json:"timestamp"` // 缓存时间戳，用于TTL
}

// CacheManager 缓存管理器结构体
type CacheManager struct {
	cache       map[string]*CacheRequest
	mutex       sync.RWMutex
	maxSize     int           // 最大缓存条目数
	ttl         time.Duration // 缓存TTL
	lastCleanup time.Time     // 上次清理时间
}

// 全局缓存管理器
var globalCacheManager *CacheManager

// 初始化缓存管理器
func init() {
	globalCacheManager = &CacheManager{
		cache:       make(map[string]*CacheRequest, 2048), // 预分配更大空间
		mutex:       sync.RWMutex{},
		maxSize:     2048,             // 最大缓存2048个条目
		ttl:         10 * time.Minute, // 10分钟TTL
		lastCleanup: time.Now(),
	}

	// 启动定期清理协程
	go globalCacheManager.startCleanupRoutine()
}

// startCleanupRoutine 启动定期清理过期缓存的协程
func (cm *CacheManager) startCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		cm.cleanupExpiredEntries()
	}
}

// cleanupExpiredEntries 清理过期的缓存条目
func (cm *CacheManager) cleanupExpiredEntries() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// 查找过期的缓存条目
	for key, entry := range cm.cache {
		if now.Sub(time.Unix(entry.Timestamp, 0)) > cm.ttl {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// 删除过期的缓存条目
	for _, key := range expiredKeys {
		delete(cm.cache, key)
	}

	cm.lastCleanup = now

	if len(expiredKeys) > 0 {
		logger.Debug(fmt.Sprintf("清理过期缓存条目 %d 个", len(expiredKeys)))
	}
}

// evictOldestEntries 驱逐最旧的缓存条目
func (cm *CacheManager) evictOldestEntries() {
	// 如果缓存未满，无需驱逐
	if len(cm.cache) < cm.maxSize {
		return
	}

	// 查找最旧的条目
	var oldestKey string
	var oldestTime = time.Now().Unix()

	for key, entry := range cm.cache {
		if entry.Timestamp < oldestTime {
			oldestTime = entry.Timestamp
			oldestKey = key
		}
	}

	// 删除最旧的条目
	if oldestKey != "" {
		delete(cm.cache, oldestKey)
		logger.Debug(fmt.Sprintf("驱逐最旧缓存条目: %s", oldestKey))
	}
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(target string, method string, followRedirects bool) string {
	return common.MD5Hash(target + ":" + method + ":" + strconv.FormatBool(followRedirects))
}

// ShouldUseCache 判断是否应该使用缓存，对于根路径的GET请求，可以重用缓存的请求和响应
func ShouldUseCache(rule finger.RuleMap, target string) (bool, CacheRequest) {
	var caches CacheRequest
	reqType := strings.ToLower(rule.Value.Request.Type)
	method := strings.ToUpper(rule.Value.Request.Method)

	// 确保是HTTP/HTTPS请求
	if reqType != "" && reqType != common.HttpType {
		return false, caches
	}

	// 只允许GET或POST请求且header为空、body为空时使用缓存
	isEmptyHeaders := len(rule.Value.Request.Headers) == 0
	isEmptyBody := rule.Value.Request.Body == ""
	isGetOrPost := method == "GET" || method == "POST"

	if !isEmptyHeaders || !isEmptyBody || !isGetOrPost {
		return false, caches
	}

	if target == "" {
		return false, caches
	}

	urlStr := common.RemoveTrailingSlash(target)
	cacheKey := GenerateCacheKey(urlStr, method, rule.Value.Request.FollowRedirects)

	logger.Debug(fmt.Sprintf("缓存提取key：%s %s %s %t", cacheKey, urlStr, method, rule.Value.Request.FollowRedirects))

	// 使用读锁访问缓存
	globalCacheManager.mutex.RLock()
	entry, exists := globalCacheManager.cache[cacheKey]
	globalCacheManager.mutex.RUnlock()

	if exists && entry != nil && entry.Request != nil && entry.Response != nil {
		// 检查缓存是否过期
		if time.Since(time.Unix(entry.Timestamp, 0)) <= globalCacheManager.ttl {
			caches.Request = entry.Request
			caches.Response = entry.Response
			return true, caches
		} else {
			// 异步删除过期缓存
			go func() {
				globalCacheManager.mutex.Lock()
				delete(globalCacheManager.cache, cacheKey)
				globalCacheManager.mutex.Unlock()
			}()
		}
	}

	return false, caches
}

// UpdateTargetCache 更新特定目标的请求响应缓存
func UpdateTargetCache(variableMap map[string]any, target string, followRedirects bool) {
	var req *proto.Request
	var resp *proto.Response

	if r, ok := variableMap["request"].(*proto.Request); ok {
		req = r
	}

	if r, ok := variableMap["response"].(*proto.Response); ok {
		resp = r
	}

	// 确保请求和响应都存在
	if req == nil || resp == nil {
		return
	}

	// 只更新结果，不需要更新缓存
	if target == "" {
		return
	}

	// 只缓存path为"/"或空、header为空、body也为空的GET或POST请求
	method := strings.ToUpper(req.Method)
	isEmptyBody := len(req.Body) == 0
	isGetOrPost := method == "GET" || method == "POST"

	if !isEmptyBody || !isGetOrPost {
		return
	}

	urlStr := common.RemoveTrailingSlash(target)
	cacheKey := GenerateCacheKey(urlStr, method, followRedirects)

	logger.Debug(fmt.Sprintf("请求缓存key：%s %s %s %t", cacheKey, urlStr, method, followRedirects))

	// 创建缓存条目（对大字段进行截断，避免缓存过大）
	const maxCacheSize = 1 << 20 // 1MB
	if len(resp.Body) > maxCacheSize {
		resp.Body = resp.Body[:maxCacheSize]
	}
	if len(resp.Raw) > maxCacheSize {
		resp.Raw = resp.Raw[:maxCacheSize]
	}
	if len(resp.RawHeader) > maxCacheSize {
		resp.RawHeader = resp.RawHeader[:maxCacheSize]
	}
	if len(req.Raw) > maxCacheSize {
		req.Raw = req.Raw[:maxCacheSize]
	}
	if len(req.Body) > maxCacheSize {
		req.Body = req.Body[:maxCacheSize]
	}

	cacheEntry := &CacheRequest{
		Request:   req,
		Response:  resp,
		Timestamp: time.Now().Unix(),
	}

	// 使用写锁更新缓存
	globalCacheManager.mutex.Lock()
	defer globalCacheManager.mutex.Unlock()

	// 检查是否需要驱逐旧缓存
	globalCacheManager.evictOldestEntries()

	// 更新缓存
	globalCacheManager.cache[cacheKey] = cacheEntry
}

// ClearTargetURLCache 删除与特定URL相关的所有缓存，无论请求方法和跟随重定向设置如何
func ClearTargetURLCache(target string) {
	if target == "" {
		return
	}

	urlStr := common.RemoveTrailingSlash(target)
	logger.Debug(fmt.Sprintf("清除URL所有缓存：%s", urlStr))

	// 预生成所有可能的缓存键
	methods := []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS"}
	redirectOptions := []bool{true, false}

	keysToDelete := make([]string, 0, len(methods)*len(redirectOptions))

	for _, method := range methods {
		for _, redirect := range redirectOptions {
			key := GenerateCacheKey(urlStr, method, redirect)
			keysToDelete = append(keysToDelete, key)
		}
	}

	// 批量删除缓存条目
	globalCacheManager.mutex.Lock()
	deletedCount := 0
	for _, key := range keysToDelete {
		if _, exists := globalCacheManager.cache[key]; exists {
			delete(globalCacheManager.cache, key)
			deletedCount++
		}
	}
	globalCacheManager.mutex.Unlock()

	if deletedCount > 0 {
		logger.Debug(fmt.Sprintf("成功删除URL相关缓存%d项：%s", deletedCount, urlStr))
	}
}

// ClearAllCache 清空所有缓存
func ClearAllCache() {
	globalCacheManager.mutex.Lock()
	// 重新初始化缓存映射
	globalCacheManager.cache = make(map[string]*CacheRequest, 2048)
	globalCacheManager.mutex.Unlock()
	logger.Debug("已清空所有缓存")
}

// GetCacheStats 获取缓存统计信息
func GetCacheStats() map[string]interface{} {
	globalCacheManager.mutex.RLock()
	defer globalCacheManager.mutex.RUnlock()

	return map[string]interface{}{
		"total_entries": len(globalCacheManager.cache),
		"max_size":      globalCacheManager.maxSize,
		"ttl_minutes":   globalCacheManager.ttl.Minutes(),
		"last_cleanup":  globalCacheManager.lastCleanup.Format(time.RFC3339),
	}
}
