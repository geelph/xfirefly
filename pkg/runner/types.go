package runner

import (
	"net/http"
	"xfirefly/pkg/finger"
	"xfirefly/pkg/types"
	"xfirefly/pkg/utils/proto"
	"xfirefly/pkg/wappalyzer"
)

// BaseInfoResponse 包含目标基础信息和HTTP响应
type BaseInfoResponse struct {
	Url        string
	Title      string
	Server     *types.ServerInfo
	StatusCode int32
	Response   *http.Response
	Wappalyzer *wappalyzer.TypeWappalyzer
	// BodyBytes 保存已读取的响应体字节，便于后续复用，避免重复读取与拷贝
	BodyBytes []byte
}

// TargetResult 存储每个目标的扫描结果
type TargetResult struct {
	URL          string                     // 目标地址
	StatusCode   int32                      // 状态码
	Title        string                     // 站点标题
	Server       *types.ServerInfo          // server信息
	Matches      []*FingerMatch             // 匹配信息
	Wappalyzer   *wappalyzer.TypeWappalyzer // 站点信息数据
	LastRequest  *proto.Request             // 该URL的请求缓存
	LastResponse *proto.Response            // 该URL的响应缓存
}

// FingerMatch 存储每个匹配的指纹信息
type FingerMatch struct {
	Finger   *finger.Finger  // 指纹信息
	Result   bool            // 识别结果
	Request  *proto.Request  // 请求数据
	Response *proto.Response // 响应数据
}

// BaseInfo 存储目标的基础信息
type BaseInfo struct {
	Title      string
	Server     *types.ServerInfo
	StatusCode int32
}

// ScanConfig 存储扫描配置参数
type ScanConfig struct {
	Proxy             string // 代理配置
	Timeout           int    // 超时配置
	URLWorkerCount    int    // 请求线程数
	FingerWorkerCount int    // 指纹检测线程数
	OutputFormat      string // 输出格式
	OutputFile        string // 输出文件
	SockOutputFile    string // 输出sock文件
}
