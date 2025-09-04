package finger

import (
	"fmt"
	"net/http"
	"strings"
	"xfirefly/pkg/network"
	"xfirefly/pkg/utils/common"
	"xfirefly/pkg/utils/proto"

	"github.com/donnie4w/go-logger/logger"
)

// formatPath 格式化路径字符串，确保路径以斜杠开头，并对特殊字符进行URL编码
// 参数:
//
//	path: 需要格式化的路径字符串，规则中的路径
//
// 返回值:
//
//	string: 格式化后的路径字符串
func formatPath(path string) string {
	newPath := strings.TrimSpace(path)
	// 如果路径以^开头，则添加斜杠
	if strings.HasPrefix(newPath, "^") {
		newPath = "/" + newPath[1:]
	}
	// 如果路径没有以斜杠开头，则添加斜杠
	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}
	// 对路径中的空格和#进行url编码
	newPath = strings.ReplaceAll(newPath, " ", "%20")
	newPath = strings.ReplaceAll(newPath, "#", "%23")
	return newPath
}

// formatBody 格式化请求体
func formatBody(body, contentType string, variableMap map[string]any) string {
	body = SetVariableMap(strings.TrimSpace(body), variableMap)
	if strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") && strings.Contains(body, "\n\n") {
		multipartBody, err := common.DealMultipart(contentType, body)
		if err != nil {
			logger.Errorf("处理multipart/form-data出错:", err)
			return body
		}
		body = SetVariableMap(strings.TrimSpace(multipartBody), variableMap)
	}
	return body
}

// buildProtoRequest 构造proto.Request结构体
func buildProtoRequest(resp *http.Response, req RuleRequest) *proto.Request {
	protoReq := &proto.Request{
		Method:      req.Method,
		Url:         network.Url2ProtoUrl(resp.Request.URL),
		ContentType: resp.Request.Header.Get("Content-Type"),
		Body:        []byte(req.Body),
	}
	headers := make(map[string]string)
	rawReqHeaderBuilder := strings.Builder{}
	for k := range resp.Request.Header {
		headers[k] = resp.Request.Header.Get(k)
		rawReqHeaderBuilder.WriteString(k)
		rawReqHeaderBuilder.WriteString(": ")
		rawReqHeaderBuilder.WriteString(resp.Request.Header.Get(k))
		rawReqHeaderBuilder.WriteString("\n")
	}

	protoReq.Headers = headers
	protoReq.Raw = []byte(fmt.Sprintf("%s %s %s\nHost: %s\n%s\n\n%s", req.Method, resp.Request.URL.Path, resp.Proto, resp.Request.URL.Host, strings.Trim(rawReqHeaderBuilder.String(), "\n"), req.Body))
	protoReq.RawHeader = []byte(strings.Trim(rawReqHeaderBuilder.String(), "\n"))

	return protoReq
}

// buildProtoResponse 构造proto.Response结构体
func buildProtoResponse(resp *http.Response, utf8RespBody string, latency int64, proxy string) *proto.Response {
	headers := make(map[string]string)
	rawHeaderBuilder := strings.Builder{}
	rawHeaderBuilder.WriteString(resp.Proto)
	rawHeaderBuilder.WriteString(" ")
	rawHeaderBuilder.WriteString(resp.Status)
	rawHeaderBuilder.WriteString("\n")
	for k := range resp.Header {
		headers[strings.ToLower(k)] = resp.Header.Get(k)
		rawHeaderBuilder.WriteString(k)
		rawHeaderBuilder.WriteString(": ")
		rawHeaderBuilder.WriteString(resp.Header.Get(k))
		rawHeaderBuilder.WriteString("\n")
	}
	// 仅在首页HTML且为GET请求时尝试解析/抓取favicon，避免在高并发下重复抓取导致内存与网络开销暴涨
	var iconHashStr = ""
	if resp.Request != nil && resp.Request.Method == http.MethodGet {
		path := resp.Request.URL.Path
		ct := resp.Header.Get("Content-Type")
		if (path == "" || path == "/") && strings.Contains(strings.ToLower(ct), "text/html") {
			iconUrl := GetIconURL(resp.Request.URL.String(), utf8RespBody)
			logger.Debugf("提取到iconUrl为: %s", iconUrl)
			iconHashStr = NewGetIconHash(iconUrl, proxy).Run()
			logger.Debugf("icon hash：%s", iconHashStr)
		}
	}
	return &proto.Response{
		Status:      int32(resp.StatusCode),
		Url:         network.Url2ProtoUrl(resp.Request.URL),
		Headers:     headers,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        []byte(utf8RespBody),
		Raw:         []byte(fmt.Sprintf("%s\n\n%s", strings.Trim(rawHeaderBuilder.String(), "\n"), utf8RespBody)),
		RawHeader:   []byte(strings.Trim(rawHeaderBuilder.String(), "\n")),
		Latency:     latency,
		IconHash:    iconHashStr,
	}
}

// BuildProtoRequest 构造proto.Request结构体 (公开版本)
func BuildProtoRequest(resp *http.Response, method, body, path string) *proto.Request {
	var req RuleRequest
	req.Method = method
	req.Body = body
	req.Path = path
	return buildProtoRequest(resp, req)
}

// BuildProtoResponse 构造proto.Response结构体 (公开版本)
func BuildProtoResponse(resp *http.Response, utf8RespBody string, latency int64, proxy string) *proto.Response {
	return buildProtoResponse(resp, utf8RespBody, latency, proxy)
}
