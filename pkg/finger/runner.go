/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: runner.go
    @Date: 2025/2/20 下午3:37*
*/
package finger

import (
	"fmt"
	"io"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
	"xfirefly/pkg/network"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
	"golang.org/x/net/context"
)

var (
	// 本地上限与 network.MaxDefaultBody 对齐，由调用方统一限制
	maxDefaultBody int64 = 512 * 1024 // 512KB
	defaultTimeout       = 5 * time.Second
)

// SendRequest yaml poc发送http请求
func SendRequest(target string, req RuleRequest, rule Rule, variableMap map[string]any, proxy string, timeout int) (map[string]any, error) {

	// 设置超时时间，如果传入的超时时间为0，则使用默认超时时间
	timeoutDuration := time.Duration(timeout) * time.Second
	if timeout <= 0 {
		timeoutDuration = defaultTimeout
	}

	options := network.OptionsRequest{
		Proxy:              "",              // 初始化为空，后面设置
		Timeout:            timeoutDuration, // 使用确定的超时参数
		Retries:            2,               // 增加重试次数
		FollowRedirects:    !rule.Request.FollowRedirects,
		InsecureSkipVerify: true, // 忽略SSL证书错误
		CustomHeaders:      map[string]string{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel() // 在读取完响应后取消

	// 设置代理地址
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			fmt.Println("代理地址解析失败:", err)
		} else {
			options.Proxy = proxyURL.String()
		}
	}

	// 处理path
	newPath := formatPath(rule.Request.Path)

	// 处理url
	urlStr := common.ParseTarget(target, newPath)

	// 处理body
	rule.Request.Body = formatBody(rule.Request.Body, rule.Request.Headers["Content-Type"], variableMap)

	// 处理自定义headers
	for k, v := range rule.Request.Headers {
		options.CustomHeaders[k] = v
	}

	// 判断请求方式
	reqType := strings.ToLower(rule.Request.Type)
	if len(reqType) > 0 && reqType != common.HttpType {
		switch reqType {
		case common.TcpType:
			rule.Request.Host = SetVariableMap(rule.Request.Host, variableMap)
			info, err := common.ParseAddress(rule.Request.Host)
			if err != nil {
				return nil, fmt.Errorf("Error parsing address: %v\n", err)
			}
			nc, err := network.NewTcpClient(rule.Request.Host, network.TcpOrUdpConfig{
				Network:     rule.Request.Type,
				ReadTimeout: time.Duration(rule.Request.ReadTimeout),
				ReadSize:    rule.Request.ReadSize,
				MaxRetries:  1,
				ProxyURL:    options.Proxy,
				IsLts:       info.IsLts,
				ServerName:  info.Host,
			})
			if err != nil {
				logger.Debug(fmt.Sprintf("tcp error：%s", err.Error()))
				return nil, err
			}
			data := rule.Request.Data

			if len(rule.Request.DataType) > 0 {
				dataType := strings.ToLower(rule.Request.DataType)
				if dataType == "hex" {
					data = common.FromHex(data)
				}
			}
			logger.Debug(fmt.Sprintf("TCP发送数据：%s", data))
			errs := nc.Send([]byte(data))
			if errs != nil {
				logger.Debug(fmt.Sprintf("tcp send error：%s", errs.Error()))
			}
			res, err := nc.RecvTcp()
			if err != nil {
				logger.Debug(fmt.Sprintf("tcp receive error：%s", err.Error()))
			}
			_ = nc.Close()
			err = network.RawParse(nc, []byte(data), res, variableMap)
			if err != nil {
				logger.Debug(fmt.Sprintf("tcp or udp parse error：%s", err.Error()))
			}
			return variableMap, nil
		case common.UdpType:
			rule.Request.Host = SetVariableMap(rule.Request.Host, variableMap)
			info, err := common.ParseAddress(rule.Request.Host)
			if err != nil {
				return nil, fmt.Errorf("Error parsing address: %v\n", err)
			}
			nc, err := network.NewUdpClient(rule.Request.Host, network.TcpOrUdpConfig{
				Network:     rule.Request.Type,
				ReadTimeout: time.Duration(rule.Request.ReadTimeout),
				ReadSize:    rule.Request.ReadSize,
				MaxRetries:  1,
				ProxyURL:    options.Proxy,
				IsLts:       info.IsLts,
				ServerName:  info.Host,
			})
			if err != nil {
				logger.Debug(fmt.Sprintf("udp error：%s", err.Error()))
				return nil, err
			}
			data := rule.Request.Data

			if len(rule.Request.DataType) > 0 {
				dataType := strings.ToLower(rule.Request.DataType)
				if dataType == "hex" {
					data = common.FromHex(data)
				}
			}
			errs := nc.Send([]byte(data))
			if errs != nil {
				fmt.Println("udp send error:", errs.Error())
			}
			res, err := nc.RecvTcp()
			if err != nil {
				fmt.Println("udp receive error:", err.Error())
			}
			_ = nc.Close()
			err = network.RawParse(nc, []byte(data), res, variableMap)
			if err != nil {
				fmt.Println("udp or udp parse error:", err.Error())
			}
			return variableMap, nil
		case common.GoType:
			fmt.Println("执行go模块调用发送请求，当前模块未完成")
			return nil, fmt.Errorf("go module not implemented")
		}
	} else {
		if len(rule.Request.Raw) > 0 {
			// 执行raw格式请求
			fmt.Println("执行raw格式请求")
			rt := network.RawHttp{RawhttpClient: network.GetRawHTTP(int(options.Timeout))}
			err := rt.RawHttpRequest(rule.Request.Raw, target, variableMap)
			if err != nil {
				return variableMap, err
			}
			return variableMap, nil
		}
	}

	// 处理协议，增加通信协议
	NewUrlStr, err := network.CheckProtocol(urlStr, options.Proxy)
	if err != nil {
		logger.Debug(fmt.Sprintf("检查http通信协议出错，错误信息：%s", err))
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			NewUrlStr = "http://" + target
		}
	}

	logger.Debug(fmt.Sprintf("请求URL：%s", NewUrlStr))

	// 发送请求
	resp, err := network.SendRequestHttp(ctx, req.Method, NewUrlStr, rule.Request.Body, options)
	if err != nil {
		logger.Debug(fmt.Sprintf("发送请求出错，错误信息：%s", err))
		return variableMap, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	newURL, err := url.Parse(NewUrlStr)
	resp.Request.URL = newURL
	// 处理请求的raw
	protoReq := buildProtoRequest(resp, rule.Request)
	variableMap["request"] = protoReq

	// 读取响应体
	reader := io.LimitReader(resp.Body, maxDefaultBody)
	body, err := io.ReadAll(reader)
	if err != nil {
		logger.Debug(fmt.Sprintf("读取响应体出错：%s", err))
		// 即使读取响应体出错，也继续处理，使用空响应体
		body = []byte{}
	}
	utf8RespBody := common.Str2UTF8(string(body))

	// 计算响应时间
	var milliseconds int64
	start := time.Now()
	trace := httptrace.ClientTrace{}
	trace.GotFirstResponseByte = func() {
		milliseconds = time.Since(start).Nanoseconds() / 1e6
	}
	// 处理响应的raw，传入代理参数
	protoResp := buildProtoResponse(resp, utf8RespBody, milliseconds, proxy)
	// 回显请求头信息
	variableMap["response"] = protoResp
	return variableMap, nil
}
