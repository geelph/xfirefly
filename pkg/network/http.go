package network

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"xfirefly/pkg/utils/common"
	"xfirefly/pkg/utils/proto"

	"github.com/chainreactors/proxyclient"
	"github.com/donnie4w/go-logger/logger"

	"github.com/zan8in/retryablehttp"
	"golang.org/x/net/context"
)

// 全局客户端配置
var (
	RetryClient    *retryablehttp.Client // 可处理重定向的客户端
	tlsConfig      *tls.Config           // tls配置
	clientInitOnce sync.Once             // 确保客户端只初始化一次
	transportCache sync.Map              // 缓存Transport对象，避免重复创建
)

// 全局客户端配置
const (
	MaxDefaultBody int64 = 512 * 1024       // 512KB
	DefaultTimeout       = 10 * time.Second // 默认请求超时时间
	HttpPrefix           = "http://"        // HTTP协议前缀
	HttpsPrefix          = "https://"       // HTTPS协议前缀
	maxRedirects         = 5                // 最大重定向次数
)

// OptionsRequest 请求配置参数结构体
type OptionsRequest struct {
	Proxy              string            // 代理地址，格式：scheme://host:port
	Timeout            time.Duration     // 请求超时时间（默认5秒）
	Retries            int               // 最大重试次数（默认3次）
	FollowRedirects    bool              // 是否跟随重定向（默认true）
	InsecureSkipVerify bool              // 是否跳过SSL证书验证（默认true）
	CustomHeaders      map[string]string // 自定义请求头
}

// 初始化全局客户端实例
func init() {
	clientInitOnce.Do(initGlobalClient)

	// 启动定期清理transport缓存的协程
	go cleanupTransportCache()
}

// initGlobalClient 初始化全局客户端实例
func initGlobalClient() {
	// 设置全局默认的TLS配置
	tlsConfig = &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}

	opts := retryablehttp.DefaultOptionsSingle
	opts.Timeout = DefaultTimeout

	transport := &http.Transport{
		TLSClientConfig:   tlsConfig,
		DisableKeepAlives: true, // 禁用连接复用，避免"Unsolicited response"错误
	}

	RetryClient = retryablehttp.NewClient(opts)
	RetryClient.HTTPClient.Transport = transport
	RetryClient.HTTPClient2.Transport = transport
}

// NewRequestHttp 创建并发送HTTP请求
func NewRequestHttp(urlStr string, options OptionsRequest) (*http.Response, error) {
	setDefaults(&options)
	if options.Proxy != "" {
		logger.Debugf("使用代理：%s", options.Proxy)
	}

	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	configureHeaders(req, options)

	client := configureClient(options)

	return client.Do(req)
}

// SendRequestHttp yaml poc or 指纹 yaml 构建发送http请求
func SendRequestHttp(ctx context.Context, Method string, UrlStr string, Body string, options OptionsRequest) (*http.Response, error) {
	setDefaults(&options)
	if options.Proxy != "" {
		logger.Debug(fmt.Sprintf("使用代理：%s", options.Proxy))
	}
	req, err := retryablehttp.NewRequestWithContext(ctx, Method, UrlStr, Body)
	if err != nil {
		return nil, err
	}
	configureHeaders(req, options)

	client := configureClient(options)

	return client.Do(req)
}

// setDefaults 设置配置参数的默认值
func setDefaults(options *OptionsRequest) {
	if options.Timeout == 0 {
		options.Timeout = 5 * time.Second
	}

	if options.Retries == 0 {
		options.Retries = 3
	}

	// 默认启用忽略TLS证书验证
	options.InsecureSkipVerify = true
}

// configureHeaders 配置请求头信息
func configureHeaders(req *retryablehttp.Request, options OptionsRequest) {
	// 设置通用请求头
	headers := map[string]string{
		"User-Agent":      common.RandomUA(),
		"Accept":          "application/x-shockwave-flash, image/gif, image/x-xbitmap, image/jpeg, image/pjpeg, application/vnd.ms-excel, application/vnd.ms-powerpoint, application/msword, */*",
		"X-Forwarded-For": common.GetRandomIP(),
		"Pragma":          "no-cache",
		"Cookie":          "cookie=" + common.RandomString(15),
		"Cache-Control":   "no-cache",
		"Connection":      "close", // 确保每次请求后不保持连接
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 默认POST内容类型
	if req.Method == http.MethodPost && req.Header.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	// 添加自定义headers
	for key, value := range options.CustomHeaders {
		req.Header.Set(key, value)
	}
}

// createTransport 创建传输层
func createTransport(proxyURL string) (*http.Transport, error) {
	// 检查缓存中是否已存在相同配置的transport
	if cachedTransport, found := transportCache.Load(proxyURL); found {
		return cachedTransport.(*http.Transport), nil
	}

	var transport *http.Transport

	if proxyURL == "" {
		transport = &http.Transport{
			TLSClientConfig:     tlsConfig,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   true, // 禁用连接复用，避免"Unsolicited response"错误
		}
	} else {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("代理地址解析失败: %v", err)
		}

		dialer, err := proxyclient.NewClient(proxy)
		if err != nil {
			return nil, fmt.Errorf("创建代理客户端失败: %v", err)
		}

		transport = &http.Transport{
			DialContext:         dialer.DialContext,
			TLSClientConfig:     tlsConfig,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   true, // 禁用连接复用，避免"Unsolicited response"错误
		}
	}

	// 存入缓存
	transportCache.Store(proxyURL, transport)

	return transport, nil
}

// configureClient 配置HTTP客户端参数
func configureClient(options OptionsRequest) *retryablehttp.Client {
	if RetryClient == nil {
		logger.Error("RetryClient 未初始化")
		initGlobalClient() // 初始化并恢复执行
	}

	// 创建新的客户端实例以避免修改全局设置
	opts := retryablehttp.DefaultOptionsSingle
	opts.Timeout = options.Timeout

	// 创建新的客户端
	client := retryablehttp.NewClient(opts)

	// 配置传输层
	transport, err := createTransport(options.Proxy)
	if err != nil {
		logger.Error("创建传输层失败: %v", err)
	} else {
		client.HTTPClient.Transport = transport
		client.HTTPClient2.Transport = transport
	}

	// 配置超时
	client.HTTPClient.Timeout = options.Timeout
	client.HTTPClient2.Timeout = options.Timeout

	// 配置重定向策略
	redirectPolicy := createRedirectPolicy(options.FollowRedirects)
	client.HTTPClient.CheckRedirect = redirectPolicy
	client.HTTPClient2.CheckRedirect = redirectPolicy

	return client
}

// createRedirectPolicy 创建重定向策略
func createRedirectPolicy(followRedirects bool) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !followRedirects {
			return http.ErrUseLastResponse // 禁止重定向
		}

		// 从之前的响应中获取Set-Cookie并添加到请求中
		if len(via) > 0 {
			for _, prevReq := range via {
				if prevReq.Response != nil && len(prevReq.Response.Header["Set-Cookie"]) > 0 {
					for _, cookie := range prevReq.Response.Cookies() {
						req.AddCookie(cookie)
					}
				}
			}
		}

		// 限制最大重定向次数
		if len(via) >= maxRedirects {
			return fmt.Errorf("达到最大重定向次数: %d", maxRedirects)
		}

		return nil
	}
}

// ReverseGet 发送GET请求并返回响应内容
func ReverseGet(target string) ([]byte, error) {
	if target == "" {
		return nil, errors.New("目标地址不能为空")
	}

	body, _, err := simpleRetryHttpGet(target, "", 0)
	return body, err
}

// simpleRetryHttpGet 简化版HTTP GET请求实现
func simpleRetryHttpGet(target string, proxy string, timeout int32) ([]byte, int, error) {
	client := RetryClient
	if client == nil {
		initGlobalClient()
		client = RetryClient
	}
	if timeout == 0 {
		timeout = 3
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", common.RandomUA())

	// 禁用重定向
	client.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// 配置传输层
	transport, err := createTransport(proxy)
	if err == nil {
		client.HTTPClient.Transport = transport
	}

	// 添加连接关闭头，确保每次请求后不保持连接
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(resp.Body)
		}
		return nil, 0, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	reader := io.LimitReader(resp.Body, MaxDefaultBody)
	respBody, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}

// CheckProtocol 检查网络通信协议
func CheckProtocol(host string, proxy string) (string, error) {
	if len(strings.TrimSpace(host)) == 0 {
		return "", fmt.Errorf("host %q is empty", host)
	}

	if strings.HasPrefix(host, HttpPrefix) || strings.HasPrefix(host, HttpsPrefix) {
		return host, nil
	}

	u, err := url.Parse(HttpPrefix + host)
	if err != nil {
		return "", err
	}

	switch u.Port() {
	case "80":
		return checkAndReturnProtocol(HttpPrefix+host, proxy)
	case "443":
		return checkAndReturnProtocol(HttpsPrefix+host, proxy)
	default:
		if result, err := checkAndReturnProtocol(HttpsPrefix+host, proxy); err == nil {
			return result, nil
		}
		return checkAndReturnProtocol(HttpPrefix+host, proxy)
	}
}
func CheckProtocolGet(target string, proxy string, timeout int) (string, error) {
	client := RetryClient
	if client == nil {
		initGlobalClient()
		client = RetryClient
	}
	if timeout == 0 {
		timeout = 3
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 使用HEAD方法代替GET，不需要读取响应体
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodHead, target, nil)
	if err != nil {
		return "", nil
	}
	req.Header.Set("User-Agent", common.RandomUA())

	// 禁用重定向
	client.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// 配置传输层
	transport, err := createTransport(proxy)
	if err == nil {
		client.HTTPClient.Transport = transport
	}

	// 添加连接关闭头，确保每次请求后不保持连接
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	//直接检查TLS状态，不读取响应体
	if resp.TLS != nil {
		return "https", nil
	}
	return "http", nil
}

func checkAndReturnProtocol(url string, proxy string) (string, error) {
	// 优化：添加协议前缀检查
	if strings.HasPrefix(url, HttpsPrefix) {
		return url, nil
	}

	// 优化：添加参数有效性检查
	if url == "" {
		return "", errors.New("URL不能为空")
	}

	res, err := CheckProtocolGet(url, proxy, 0)
	if err != nil {
		// 优化：添加更具体的错误信息
		return "", fmt.Errorf("检查协议失败: %w", err)
	}

	// 判断是否为HTTPS协议
	if res == "https" {
		// 只有当URL以HTTP前缀开始时才进行替换
		if strings.HasPrefix(url, HttpPrefix) {
			return HttpsPrefix + url[len(HttpPrefix):], nil
		}
		// 如果没有前缀，则添加HTTPS前缀
		return HttpsPrefix + url, nil
	}

	// 如果是HTTP协议但没有前缀，则添加HTTP前缀
	if !strings.HasPrefix(url, HttpPrefix) {
		return HttpPrefix + url, nil
	}

	return url, nil
}

// Url2ProtoUrl URL转换为Proto URL
func Url2ProtoUrl(u *url.URL) *proto.UrlType {
	return &proto.UrlType{
		Scheme:   u.Scheme,
		Domain:   u.Hostname(),
		Host:     u.Host,
		Port:     u.Port(),
		Path:     u.EscapedPath(),
		Query:    u.RawQuery,
		Fragment: u.Fragment,
	}
}

// ParseRequest 解析请求raw数据包
func ParseRequest(oReq *http.Request) (*proto.Request, error) {
	req := &proto.Request{
		Method: oReq.Method,
		Url:    common.Url2UrlType(oReq.URL),
	}

	// 提取请求头
	header := make(map[string]string, len(oReq.Header))
	for k := range oReq.Header {
		header[k] = oReq.Header.Get(k)
	}
	req.Headers = header
	req.ContentType = oReq.Header.Get("Content-Type")

	// 提取请求体
	if oReq.Body != nil && oReq.Body != http.NoBody {
		data, err := io.ReadAll(oReq.Body)
		if err != nil {
			return nil, err
		}
		req.Body = data
		oReq.Body = io.NopCloser(bytes.NewBuffer(data))
	}

	return req, nil
}

// cleanupTransportCache 定期清理transport缓存
func cleanupTransportCache() {
	ticker := time.NewTicker(5 * time.Minute) // 更频繁清理
	defer ticker.Stop()

	for range ticker.C {
		// 遍历所有缓存的transport并关闭空闲连接
		transportCache.Range(func(key, value interface{}) bool {
			if transport, ok := value.(*http.Transport); ok {
				transport.CloseIdleConnections()
			}
			return true
		})

		// 记录清理日志
		logger.Debug("已清理transport缓存中的空闲连接")

		// 如果缓存过大，可以考虑完全重置
		var count int
		transportCache.Range(func(_, _ interface{}) bool {
			count++
			return true
		})
		// 如果缓存项超过50个，重置缓存
		if count > 100 {
			transportCache = sync.Map{}
			logger.Debug("transport缓存过大，已重置")
		}
	}
}
