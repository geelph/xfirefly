/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: server.go
    @Date: 2025/4/3 上午10:10*
*/
package finger

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"xfirefly/pkg/types"
)

// ExtractServerInfo 从HTTP响应头中提取server信息
func ExtractServerInfo(header http.Header) (string, string) {
	serverValue := header.Get("Server")
	if serverValue == "" {
		return "", ""
	}

	// 提取版本 - 先处理，因为cleanServerString可能会移除版本信息
	version := ExtractVersion(serverValue)

	// 清理无用内容
	cleanedServer := CleanServerString(serverValue)

	// 移除服务器名称后的版本号（如 nginx/1.18.0 变为 nginx）
	if version != "" {
		slashVersionPattern := regexp.MustCompile(`\/` + regexp.QuoteMeta(version))
		cleanedServer = slashVersionPattern.ReplaceAllString(cleanedServer, "")
	}

	// 组合服务器信息 - 保留原始格式（包括版本号）
	if version != "" && !strings.Contains(cleanedServer, version) {
		if strings.Contains(serverValue, "/"+version) {
			cleanedServer = strings.Replace(serverValue, " ("+version+")", "", -1)
			cleanedServer = strings.Replace(cleanedServer, "("+version+")", "", -1)
		}
	}

	// 最后清理可能剩余的多余空格
	cleanedServer = strings.TrimSpace(cleanedServer)

	return cleanedServer, version
}

// CleanServerString 移除服务器信息中没有用的内容
func CleanServerString(server string) string {
	// 移除括号及其内容
	noParentheses := regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(server, "")

	// 移除多余空格
	trimmed := strings.TrimSpace(noParentheses)

	// 处理常见的无用修饰词
	cleaned := strings.ReplaceAll(trimmed, "powered by ", "")
	cleaned = strings.ReplaceAll(cleaned, "running on ", "")

	return cleaned
}

// ExtractVersion 从服务器字符串中提取版本信息
func ExtractVersion(server string) string {
	// 检查是否有 name/version 格式
	slashVersionRegex := regexp.MustCompile(`\/(\d+(\.\d+)*)`)
	slashMatches := slashVersionRegex.FindStringSubmatch(server)
	if len(slashMatches) > 1 {
		return slashMatches[1]
	}

	// 尝试匹配括号中的版本号，如 Server (1.2.3)
	parenthesisVersionRegex := regexp.MustCompile(`\(.*?(\d+\.\d+(\.\d+)*).*?\)`)
	parenthesisMatches := parenthesisVersionRegex.FindStringSubmatch(server)
	if len(parenthesisMatches) > 1 {
		return parenthesisMatches[1]
	}

	// 尝试匹配任何可能的版本格式
	versionRegex := regexp.MustCompile(`(\d+\.\d+(\.\d+)?)`)
	matches := versionRegex.FindStringSubmatch(server)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// FormatServerResult 格式化显示服务器信息结果
func FormatServerResult(originalServer, cleanedServer, version string) string {
	var result strings.Builder

	// 原始服务器信息
	result.WriteString(fmt.Sprintf("原始服务器信息: %s\n", originalServer))

	// 清理后的服务器信息
	if cleanedServer != "" {
		result.WriteString(fmt.Sprintf("服务器类型: %s\n", cleanedServer))
	} else {
		result.WriteString("服务器类型: 未知\n")
	}

	// 版本信息
	if version != "" {
		result.WriteString(fmt.Sprintf("版本号: %s\n", version))
	} else {
		result.WriteString("版本号: 未知\n")
	}

	return result.String()
}

// GetServerInfoFromResponse 从HTTP响应中获取并格式化服务器信息
// 返回ServerInfo结构体指针
func GetServerInfoFromResponse(resp *http.Response) *types.ServerInfo {
	if resp == nil {
		// 如果响应为空，返回空的ServerInfo
		return types.EmptyServerInfo()
	}

	originalServer := resp.Header.Get("Server")
	if originalServer == "" {
		// 如果没有Server头，返回空的ServerInfo
		return types.EmptyServerInfo()
	}

	cleanedServer, version := ExtractServerInfo(resp.Header)

	// 创建并返回ServerInfo对象
	return types.NewServerInfo(originalServer, cleanedServer, version)
}

// GetServerInfoFromTCP 从TCP/UDP响应中获取并格式化服务器信息
// 返回ServerInfo结构体指针
func GetServerInfoFromTCP(address, hostType string) *types.ServerInfo {
	// 创建并返回ServerInfo对象
	return types.NewServerInfo(
		fmt.Sprintf("%s %s", hostType, address),
		hostType,
		"未知",
	)
}
