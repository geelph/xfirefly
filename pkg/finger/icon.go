package finger

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"xfirefly/pkg/network"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
	"github.com/spaolacci/murmur3"
	_ "github.com/vmihailenco/msgpack/v5"
)

// GetIconHash 获取icon hash
type GetIconHash struct {
	iconURL    string
	retries    int
	headers    map[string]string
	fileHeader []string
	proxy      string
}

// NewGetIconHash 初始化 GetIconHash
func NewGetIconHash(iconURL string, proxy string, retries ...int) *GetIconHash {
	// 设置默认值为 1
	retriesValue := 1
	if len(retries) > 0 {
		retriesValue = retries[0]
	}

	return &GetIconHash{
		iconURL: iconURL,
		retries: retriesValue,
		headers: map[string]string{
			"User-Agent":      common.RandomUA(),
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.5",
			"Connection":      "close",
		},
		fileHeader: []string{
			"89504E470", "89504e470", "00000100", "474946383", "FFD8FFE00", "FFD8FFE10", "3c7376672", "3c3f786d6",
		},
		proxy: proxy,
	}
}

// getDefaultIconURL 获取默认的icon URL
// return: http://xxx.com/favicon.ico
func (g *GetIconHash) getDefaultIconURL(iconURL string) string {
	if iconURL == "" {
		return ""
	}
	parsedURL, err := url.Parse(iconURL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s://%s/favicon.ico", parsedURL.Scheme, parsedURL.Host)
}

// getIconHash calculates the hash value of an icon based on its URL.
// It handles both data URLs (base64 encoded images) and HTTP URLs.
//
// Parameters:
//
//	iconURL - the URL of the icon, either a data URL or HTTP URL
//
// Returns:
//
//	int32 - the calculated hash value of the icon
func (g *GetIconHash) getIconHash(iconURL string) int32 {
	// Check if the icon URL is a data URL (base64 encoded image)
	if strings.HasPrefix(iconURL, "data:") {
		return g.hashDataURL(iconURL)
	}
	// Handle HTTP URLs
	return g.hashHTTPURL(iconURL)
}

// hashDataURL 处理 data URL 并计算 hash 值
func (g *GetIconHash) hashDataURL(iconURL string) int32 {
	parts := strings.Split(iconURL, ",")
	if len(parts) != 2 {
		return 0
	}
	//iconData := StandBase64([]byte(parts[1]))
	//if len(iconData) != 0 {
	//	return Mmh3Hash32(iconData)
	//}
	// 修复+被意外转为%20（前面获取是按照iconurl进行的操作）
	base64Part := strings.ReplaceAll(parts[1], "%20", "+")
	//logger.Info(base64Part)
	iconData, err := base64.StdEncoding.DecodeString(base64Part)
	if err != nil {
		// 处理错误，比如日志或返回
		logger.Warnf("Base64 decode failed:", err)
		return 0
	}
	return Mmh3Hash32(StandBase64(iconData))
	//return 0
}

// hashHTTPURL 处理 HTTP URL 并计算 hash 值
func (g *GetIconHash) hashHTTPURL(iconURL string) int32 {
	options := network.OptionsRequest{
		Proxy:              g.proxy,
		Timeout:            5 * time.Second,
		Retries:            2,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
		CustomHeaders:      g.headers,
	}
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// 发送请求
	resp, err := network.SendRequestHttp(ctx, "GET", iconURL, "", options)
	if err != nil {
		logger.Debugf("创建请求失败: %s", err)
		return 0
	}

	// 读取响应体（限制最大1MB）
	var bodyBytes []byte
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(io.LimitReader(resp.Body, network.MaxDefaultBody))
		if err != nil {
			logger.Debugf("读取响应体失败: %s", err)
			return 0
		}
		defer func() { _ = resp.Body.Close() }()

		// 验证是否为图片
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "image") && len(bodyBytes) > 0 {
			return Mmh3Hash32(StandBase64(bodyBytes))
		}

		if len(bodyBytes) > 0 {
			bodyHex := fmt.Sprintf("%x", bodyBytes[:8])
			logger.Debugf("响应头前8个字节: %s", bodyHex)
			for _, fh := range g.fileHeader {
				if strings.HasPrefix(bodyHex, strings.ToLower(fh)) {
					return Mmh3Hash32(StandBase64(bodyBytes))
				}
			}
		}
	}

	return 0
}

// StandBase64 标准化Base64编码
func StandBase64(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte{}
	}
	bckd := base64.StdEncoding.EncodeToString(raw)
	var buffer bytes.Buffer
	for i := 0; i < len(bckd); i++ {
		ch := bckd[i]
		buffer.WriteByte(ch)
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')
	return buffer.Bytes()
}

// Mmh3Hash32 计算Mmh3Hash32哈希值
func Mmh3Hash32(raw []byte) int32 {
	hasher := murmur3.New32()
	_, _ = hasher.Write(raw)
	return int32(hasher.Sum32())
}

// Run 运行获取icon hash的流程
func (g *GetIconHash) Run() string {
	var hash int32
	if g.iconURL != "" {
		hash = g.getIconHash(g.iconURL)
	}
	if hash == 0 {
		// 浏览器访问会发送一个默认的icon请求
		defaultURL := g.getDefaultIconURL(g.iconURL)
		if defaultURL != "" {
			hash = g.getIconHash(defaultURL)
		}
	}
	return fmt.Sprintf("%d", hash)
}

// GetIconURL 获取icon的url地址
func GetIconURL(pageURL string, html string) string {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		logger.Error(fmt.Sprintf("URL解析错误: %s", err))
		return ""
	}

	baseURL := fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host)
	basePath := parsedURL.Path
	if strings.Contains(basePath, ".") || strings.Contains(basePath, ".htm") {
		basePath = ""
	}

	// 默认favicon.ico路径
	faviconURL := baseURL + "favicon.ico"

	// 检查HTML中是否有icon标签
	htmlLower := strings.ToLower(html)

	// 查找所有可能的icon标签
	iconTags := []string{
		"<link rel=\"icon\"",
		"<link rel='icon'",
		"<link rel=icon",
		"<link rel=\"shortcut icon\"",
		"<link rel=shortcut icon",
		"<link type=\"image/x-icon\"",
		"<link type=image/x-icon",
		"<link rel=\"apple-touch-icon\"",
		"<link rel=\"apple-touch-icon-precomposed\"",
		"<link id=\"favicon\"",
		"<link id=favicon",
		"<link rel=\"fluid-icon\"",
		"<link rel=\"mask-icon\"",
		"<link rel=\"alternate icon\"",
		"<link rel=\"apple-touch-startup-image\"",
		"<link rel=\"apple-touch-icon-image\"",
		"<link rel=\"icon shortcut\"",
		"<link rel=icon shortcut",
		"<link rel=\"msapplication-TileImage\"",
		"<link rel=\"msapplication-square70x70logo\"",
		"<link rel=\"msapplication-square150x150logo\"",
		"<link rel=\"msapplication-wide310x150logo\"",
		"<link rel=\"msapplication-square310x310logo\"",
		"<link rel=\"msapplication-config\"",
		"<link rel=\"shortcut\"",
		"<link rel=\"manifest\"",
		"<meta name=\"msapplication-TileImage\"",
		"<meta property=\"og:image\"",
		"<meta itemprop=\"image\"",
		"<meta itemprop=image",
	}

	// 按照优先级排序的icon路径
	var candidateIcons []string

	// 寻找所有匹配的icon标签
	for _, tag := range iconTags {
		startPos := 0
		for {
			index := strings.Index(htmlLower[startPos:], tag)
			if index == -1 {
				break
			}

			tagStartIndex := startPos + index
			tagEnd := strings.Index(html[tagStartIndex:], ">") + tagStartIndex
			if tagEnd > tagStartIndex {
				linkTag := html[tagStartIndex:tagEnd]

				// 提取href或content属性
				reAttr := regexp.MustCompile(`(?:href|content)=["']?([^"'>\s]+)`)
				attrMatch := reAttr.FindStringSubmatch(linkTag)
				if len(attrMatch) > 1 {
					iconPath := attrMatch[1]

					// 检查是否为常见图片格式或包含关键词
					if isImagePath(iconPath) ||
						(strings.Contains(linkTag, "icon") ||
							strings.Contains(linkTag, "favicon") ||
							strings.Contains(linkTag, "logo") ||
							strings.Contains(linkTag, "image")) {
						candidateIcons = append(candidateIcons, iconPath)
					}
				}
			}
			startPos = tagStartIndex + 1
			if startPos >= len(htmlLower) {
				break
			}
		}
	}

	// 根据可能的icon标签没找到，尝试从link标签中寻找ico图标
	if len(candidateIcons) == 0 {
		// 查找link标签中可能的favicon
		reIcon := regexp.MustCompile(`<link[^>]+href=["']([^"']+\.ico)`)
		iconList := reIcon.FindAllStringSubmatch(html, -1)
		for _, match := range iconList {
			if len(match) > 1 {
				candidateIcons = append(candidateIcons, match[1])
			}
		}
	}

	// 查找所有图片标签中可能的favicon
	reImg := regexp.MustCompile(`<img[^>]+src=["']([^"']+(?:favicon|icon)[^"']*)["'][^>]*>`)
	imgMatches := reImg.FindAllStringSubmatch(html, -1)
	for _, match := range imgMatches {
		if len(match) > 1 {
			candidateIcons = append(candidateIcons, match[1])
		}
	}

	// 如果没有找到标准icon标签，尝试查找所有可能的图标链接
	if len(candidateIcons) == 0 {
		re := regexp.MustCompile(`href=["']([^"']+\.(ico|png|jpg|jpeg|gif|svg|webp))["']`)
		iconList := re.FindAllStringSubmatch(html, -1)

		for _, match := range iconList {
			if len(match) > 1 {
				candidateIcons = append(candidateIcons, match[1])
			}
		}
	}

	// 优化：直接使用map存储清理后的URL和原始URL的对应关系
	iconMap := make(map[string]string)
	for _, icon := range candidateIcons {
		// 使用url包处理URL
		cleaned := icon
		if parsedURL, err := url.Parse(icon); err == nil {
			// 清除查询参数
			parsedURL.RawQuery = ""
			// 重新构建URL字符串
			cleaned = parsedURL.String()
		}

		// 如果已经存在相同的清理后URL，保留原始URL
		if _, exists := iconMap[cleaned]; !exists {
			iconMap[cleaned] = icon
		}
	}

	// 将map转换为切片以便排序
	var sortedIcons []string
	for cleaned := range iconMap {
		sortedIcons = append(sortedIcons, cleaned)
	}

	// 对清理后的URL进行排序
	sort.Slice(sortedIcons, func(i, j int) bool {
		iIsIco := strings.HasSuffix(strings.ToLower(sortedIcons[i]), ".ico")
		jIsIco := strings.HasSuffix(strings.ToLower(sortedIcons[j]), ".ico")

		// 如果一个是.ico而另一个不是，.ico的排在前面
		if iIsIco != jIsIco {
			return iIsIco
		}

		// 如果都是.ico或都不是.ico，保持原有顺序
		return i < j
	})

	// 将排序后的原始URL重新放回candidateIcons
	candidateIcons = candidateIcons[:0] // 清空切片但保留容量
	for _, cleaned := range sortedIcons {
		candidateIcons = append(candidateIcons, iconMap[cleaned])
	}

	for _, iconPath := range candidateIcons {
		absoluteURL := buildAbsoluteURL(parsedURL, baseURL, basePath, iconPath)
		if absoluteURL != "" {
			normalized := normalizeFaviconURL(absoluteURL)
			logger.Debug(fmt.Sprintf("找到可能的icon url: %s", normalized))
			return normalized

		}
	}

	// 如果没有找到有效的图标，返回默认favicon
	defaultURL := normalizeFaviconURL(faviconURL)

	return defaultURL
}

// buildAbsoluteURL 构建绝对URL
func buildAbsoluteURL(parsedURL *url.URL, baseURL, basePath, iconPath string) string {
	// 跳过空路径
	if iconPath == "" {
		return ""
	}

	// 已经是完整URL
	if strings.HasPrefix(iconPath, "http://") || strings.HasPrefix(iconPath, "https://") {
		return iconPath
	}

	// 跳过data:URL
	if strings.HasPrefix(iconPath, "data:") {
		return iconPath
	}

	// 协议相对URL
	if strings.HasPrefix(iconPath, "//") {
		return parsedURL.Scheme + ":" + iconPath
	}

	// 尝试使用标准库解析相对URL
	relURL, err := url.Parse(iconPath)
	if err == nil {
		absURL := parsedURL.ResolveReference(relURL)
		if absURL.String() != "" {
			return absURL.String()
		}
	}

	// 绝对路径
	if strings.HasPrefix(iconPath, "/") {
		return baseURL + strings.TrimPrefix(iconPath, "/")
	}

	// 相对路径
	if basePath == "" || strings.HasSuffix(basePath, "/") {
		return baseURL + strings.TrimPrefix(basePath, "/") + iconPath
	}

	// 基路径不以/结尾，需要获取目录部分
	dir := path.Dir(basePath)
	if dir == "." {
		dir = ""
	} else if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return baseURL + strings.TrimPrefix(dir, "/") + iconPath
}

// normalizeFaviconURL 规范化favicon URL
func normalizeFaviconURL(url string) string {
	if url == "" {
		return ""
	}

	// 处理URL编码问题
	decodedURL, err := common.URLDecode(url)
	if err == nil && decodedURL != url {
		url = decodedURL
	}

	// 修复双斜杠问题，但保留协议中的双斜杠
	result := url
	if strings.HasPrefix(result, "http://") {
		result = "http://" + strings.ReplaceAll(result[7:], "//", "/")
	} else if strings.HasPrefix(result, "https://") {
		result = "https://" + strings.ReplaceAll(result[8:], "//", "/")
	}

	// 处理特殊字符 - 一次性替换所有字符
	replacer := strings.NewReplacer(
		" ", "%20",
		"\"", "%22",
		"'", "%27",
		"<", "%3C",
		">", "%3E",
	)
	result = replacer.Replace(result)

	// 移除URL中的锚点
	if idx := strings.Index(result, "#"); idx != -1 {
		result = result[:idx]
	}

	// 确保URL格式正确
	if !strings.HasPrefix(result, "http://") && !strings.HasPrefix(result, "https://") && !strings.HasPrefix(result, "data:") {
		// 尝试添加协议
		if strings.HasPrefix(result, "//") {
			result = "https:" + result
		} else {
			result = "https://" + result
		}
	}

	return result
}

// isImagePath 检查路径是否为图片格式
func isImagePath(path string) bool {
	lowerPath := strings.ToLower(path)
	extensions := []string{".ico", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp"}

	for _, ext := range extensions {
		if strings.HasSuffix(lowerPath, ext) {
			return true
		}
	}
	return false
}
