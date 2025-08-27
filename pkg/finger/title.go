/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: title.go
    @Date: 2025/4/3 上午9:47*
*/
package finger

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
)

// GetTitle 从网页中提取标题
func GetTitle(urlStr string, resp *http.Response) string {
	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Debug("读取响应体出错: %v", err)
		return ""
	}
	// 不要忘记恢复响应体以便后续使用
	resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// 解析字符集并转换编码
	bodyText := string(bodyBytes)
	contentType := resp.Header.Get("Content-Type")

	// 检查和处理编码
	charsetRegex := regexp.MustCompile(`(?i)charset=["']?([\w-]+)["']?`)
	charsetMatch := charsetRegex.FindStringSubmatch(contentType)
	if len(charsetMatch) < 2 {
		// 如果 HTTP 头中没有指定字符集，尝试从 HTML 内容中查找
		metaCharsetRegex := regexp.MustCompile(`(?i)<meta\s+.*?charset=["']?([\w-]+)["']?.*?>`)
		metaMatch := metaCharsetRegex.FindStringSubmatch(bodyText)
		if len(metaMatch) >= 2 {
			charsetMatch = metaMatch
		}
	}

	// 根据检测到的字符集进行转换
	if len(charsetMatch) >= 2 {
		charset := strings.ToLower(charsetMatch[1])
		logger.Debug("检测到字符集: %s", charset)

		if charset != "utf-8" && charset != "utf8" {
			// 使用 common.Str2UTF8 函数转换为 UTF-8
			bodyText = common.Str2UTF8(bodyText)
			logger.Debug("已将内容从 %s 转换为 UTF-8", charset)
		}
	} else {
		// 如果无法检测到字符集，尝试转换为 UTF-8
		bodyText = common.Str2UTF8(bodyText)
	}

	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		logger.Debug("解析URL出错: %v", err)
		return ""
	}

	// 获取基础URL
	baseURL := fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host)
	basePath := parsedURL.Path

	var title string
	var titleURL string

	// 使用正则表达式查找标题，使用(?s)模式修饰符支持跨行匹配
	titleRegex := regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	titleMatches := titleRegex.FindStringSubmatch(bodyText)
	if len(titleMatches) > 1 {
		title = cleanTitle(titleMatches[1])
		logger.Debug("通过正则表达式识别到标题: %s", title)
	}

	// 在JavaScript中查找document.title
	domTitleRegex := regexp.MustCompile(`(?i)document\.title.*?=.*?\((.*?)\)`)
	domTitleMatches := domTitleRegex.FindStringSubmatch(bodyText)
	if len(domTitleMatches) > 1 {
		logger.Debug("识别到DOM渲染的标题: %s", domTitleMatches[1])
		domTitle := strings.ReplaceAll(domTitleMatches[1], "\"", "")

		invalidTitles := []string{"title", ".title", "top.", ".login", "=", "||", "''", "null"}
		isInvalid := false
		for _, invalid := range invalidTitles {
			if strings.Contains(domTitle, invalid) {
				isInvalid = true
				break
			}
		}
		if !isInvalid && len(domTitle) > 0 {
			lowerDomTitle := strings.ToLower(domTitle)
			if !strings.Contains(lowerDomTitle, "null") && !strings.Contains(lowerDomTitle, "--") && !strings.Contains(title, ".title") && !strings.Contains(title, "document") && len(title)-len(domTitle) > 30 {
				logger.Debug("DOM标题符合要求，更新标题")
				title = domTitle
			} else {
				logger.Debug("DOM标题不符合要求，跳过")
			}
		} else {
			logger.Debug("DOM标题不符合要求，跳过")
		}

	}

	// 查找i18n JavaScript文件
	i18nRegex := regexp.MustCompile(`(?i)type="text/javascript".*?src="(.*?)"`)
	i18nMatches := i18nRegex.FindAllStringSubmatch(bodyText, -1)

	for _, match := range i18nMatches {
		if len(match) > 1 && strings.HasSuffix(match[1], ".js") && strings.Contains(match[1], "i18n") {
			path := strings.TrimPrefix(match[1], "/")
			if strings.HasPrefix(path, basePath) {
				titleURL = baseURL + path
			} else {
				titleURL = baseURL + strings.TrimSuffix(basePath, "/") + "/" + path
			}
			break
		}
	}

	// 尝试从i18n JavaScript文件获取标题
	if titleURL != "" {
		logger.Debug("识别到国际化，从i18n JS文件获取标题数据")

		retries := 3
		for i := 0; i < retries; i++ {
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return nil // 允许重定向
				},
			}

			req, err := http.NewRequest("GET", titleURL, nil)
			if err != nil {
				logger.Debug("创建请求出错: %v", err)
				break
			}

			// 从原始请求复制头信息
			for k, v := range resp.Request.Header {
				req.Header[k] = v
			}

			respTitle, err := client.Do(req)
			if err != nil {
				logger.Debug("获取i18n JS文件出错: %v", err)
				continue
			}

			if respTitle.StatusCode == 200 {
				bodyBytes, err := io.ReadAll(respTitle.Body)
				_ = respTitle.Body.Close()
				if err != nil {
					logger.Debug("读取i18n JS响应出错: %v", err)
					continue
				}

				// 将 JS 文件内容转换为 UTF-8
				jsContent := common.Str2UTF8(string(bodyBytes))

				titleRegex := regexp.MustCompile(`"top\.login\.title": "(.*?)",`)
				titleMatches := titleRegex.FindStringSubmatch(jsContent)
				if len(titleMatches) > 1 {
					logger.Debug("成功从i18n JS文件获取标题数据: %s", titleMatches[1])
					title = titleMatches[1]
					logger.Debug("找到新标题，替换原始标题: %s", title)
				}
			}
			break
		}
	}

	return title
}

// cleanTitle 移除空白字符并清理标题字符串
func cleanTitle(title string) string {
	// 先确保标题是UTF-8编码
	title = common.Str2UTF8(title)

	// 移除制表符、换行符和回车符
	title = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\t' {
			return ' ' // 将这些字符替换为空格，而不是删除它们
		}
		return r
	}, title)

	// 将多个空格替换为单个空格
	space := false
	title = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			if space {
				return -1
			}
			space = true
			return ' '
		}
		space = false
		return r
	}, title)

	return strings.TrimSpace(title)
}
