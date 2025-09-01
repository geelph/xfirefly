package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"xfirefly/pkg/finger"
	"xfirefly/pkg/network"
	"xfirefly/pkg/types"
	"xfirefly/pkg/utils/common"
	"xfirefly/pkg/utils/proto"
	"xfirefly/pkg/wappalyzer"

	"github.com/donnie4w/go-logger/logger"
)

// initializeCache 基于基础信息构建初始 Request/Response，避免重复读取响应体
func initializeCache(base *BaseInfoResponse, proxy string) (*proto.Response, *proto.Request) {
	if base == nil || base.Response == nil {
		return nil, nil
	}
	httpResp := base.Response

	// 优先使用已缓存的 BodyBytes，避免重复 ReadAll
	respBody := base.BodyBytes
	if respBody == nil {
		data, err := io.ReadAll(httpResp.Body)
		if err != nil {
			logger.Debug(fmt.Sprintf("读取响应体出错: %v", err))
			data = []byte{}
		}
		respBody = data
	}
	// 重置响应体（供后续使用）
	httpResp.Body = io.NopCloser(bytes.NewReader(respBody))

	utf8RespBody := common.Str2UTF8(string(respBody))

	// 构建响应/请求对象
	initialResponse := finger.BuildProtoResponse(httpResp, utf8RespBody, 0, proxy)
	initialRequest := finger.BuildProtoRequest(httpResp, "GET", "", "/")
	return initialResponse, initialRequest
}

// GetBaseInfo 获取目标的基础信息并返回 BaseInfoResponse 结构体
func GetBaseInfo(target, proxy string, timeout int) (*BaseInfoResponse, error) {
	// 检查并规范化URL协议
	if checkedURL, err := network.CheckProtocol(target, proxy); err == nil && checkedURL != "" {
		target = checkedURL
	}
	logger.Debug(fmt.Sprintf("请求协议修正后url: %s", target))
	// 二次验证URL
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}
	// 设置超时时间
	timeoutDuration := time.Duration(timeout) * time.Second
	if timeout <= 0 {
		timeoutDuration = 5 * time.Second
	}

	// 创建请求选项
	options := network.OptionsRequest{
		Proxy:              proxy,
		Timeout:            timeoutDuration,
		Retries:            3,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
	}

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	resp, err := network.SendRequestHttp(ctx, "GET", target, "", options)
	if err != nil {
		return &BaseInfoResponse{
			Url:        target,
			Title:      "",
			Server:     types.EmptyServerInfo(),
			StatusCode: 0,
			Response:   resp,
			Wappalyzer: nil,
			BodyBytes:  nil,
		}, fmt.Errorf("发送请求失败: %v", err)
	}

	// 提取基本信息
	statusCode := int32(resp.StatusCode)
	title := finger.GetTitle(target, resp)
	serverInfo := finger.GetServerInfoFromResponse(resp)
	newURL, _ := url.Parse(target)
	if resp.Request != nil {
		resp.Request.URL = newURL
	}

	// 获取站点技术信息
	wapp, err := wappalyzer.NewWappalyzer()
	if err != nil {
		// 即使获取站点技术信息失败，仍然返回基本信息
		return &BaseInfoResponse{
			Url:        target,
			Title:      title,
			Server:     serverInfo,
			StatusCode: statusCode,
			Response:   resp,
			Wappalyzer: nil,
		}, nil
	}
	// 读取响应体一次并保存，后续复用（限制大小，避免大包体导致内存暴涨）
	data, err := io.ReadAll(io.LimitReader(resp.Body, network.MaxDefaultBody))
	if err != nil {
		logger.Debugf("读取响应体出错: %v", err)
		data = []byte{}
	}
	// 重置响应体以供后续使用
	resp.Body = io.NopCloser(bytes.NewReader(data))

	wappData, err := wapp.GetWappalyzer(resp.Header, data)
	if err != nil {
		// 即使获取Wappalyzer数据失败，仍然返回基本信息
		return &BaseInfoResponse{
			Url:        target,
			Title:      title,
			Server:     serverInfo,
			StatusCode: statusCode,
			Response:   resp,
			Wappalyzer: nil,
		}, nil
	}

	logger.Debugf("当前站点使用技术：%s", wappData)

	return &BaseInfoResponse{
		Url:        target,
		Title:      title,
		Server:     serverInfo,
		StatusCode: statusCode,
		Response:   resp,
		Wappalyzer: wappData,
		BodyBytes:  data,
	}, nil
}
