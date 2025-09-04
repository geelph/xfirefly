package common

import (
	"bytes"
	"crypto/md5"
	rand2 "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"xfirefly/pkg/utils/proto"

	"github.com/axgle/mahonia"
	"github.com/donnie4w/go-logger/logger"
	"github.com/spaolacci/murmur3"
)

// AddressInfo 封装解析结果的结构体
type AddressInfo struct {
	Host     string
	Port     string
	Scheme   string
	Hostname string
	IsLts    bool
}

const (
	HttpType = "http"
	TcpType  = "tcp"
	UdpType  = "udp"
	SslType  = "ssl"
	GoType   = "go"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"
const letterNumberBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const lowletterNumberBytes = "0123456789abcdefghijklmnopqrstuvwxyz"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// 全局变量
var (
	// 维护的日志等级变量，go-logger中没有提供GetLevel的函数
	LogLevel = logger.LEVEL_INFO
)

var compoundDomains = map[string]bool{
	"ac.uk":  true,
	"co.uk":  true,
	"gov.uk": true,
	"ltd.uk": true,
	"me.uk":  true,
	"net.au": true,
	"org.au": true,
	"com.au": true,
	"edu.au": true,
	"gov.au": true,
	"asn.au": true,
	"id.au":  true,
	"com.cn": true,
	"net.cn": true,
	"org.cn": true,
	"gov.cn": true,
	"edu.cn": true,
	"mil.cn": true,
	"ac.cn":  true,
	"ah.cn":  true,
	"bj.cn":  true,
	"cq.cn":  true,
	"fj.cn":  true,
	"gd.cn":  true,
	"gs.cn":  true,
	"gx.cn":  true,
	"gz.cn":  true,
	"ha.cn":  true,
	"hb.cn":  true,
	"he.cn":  true,
	"hi.cn":  true,
	"hl.cn":  true,
	"hn.cn":  true,
	"jl.cn":  true,
	"js.cn":  true,
	"jx.cn":  true,
	"ln.cn":  true,
	"nm.cn":  true,
	"nx.cn":  true,
	"qh.cn":  true,
	"sc.cn":  true,
	"sd.cn":  true,
	"sh.cn":  true,
	"sn.cn":  true,
	"sx.cn":  true,
	"tj.cn":  true,
	"xj.cn":  true,
	"xz.cn":  true,
	"yn.cn":  true,
	"zj.cn":  true,
}

// GetRootDomain 获取域名的根域名
func GetRootDomain(input string) (string, error) {
	input = strings.TrimLeft(input, "http://")
	input = strings.TrimLeft(input, "https://")
	input = strings.Trim(input, "/")
	ip := net.ParseIP(input)
	if ip != nil {
		return ip.String(), nil
	}
	input = "https://" + input

	// 尝试解析为 URL
	u, err := url.Parse(input)
	if err == nil && u.Hostname() != "" {
		ipHost := net.ParseIP(u.Hostname())
		if ipHost != nil {
			return ipHost.String(), nil
		}
		hostParts := strings.Split(u.Hostname(), ".")
		if len(hostParts) < 2 {
			return "", fmt.Errorf("域名格式不正确")
		}
		if len(hostParts) == 2 {
			return u.Hostname(), nil
		}
		// 检查是否为复合域名
		if _, ok := compoundDomains[hostParts[len(hostParts)-2]+"."+hostParts[len(hostParts)-1]]; ok {
			return hostParts[len(hostParts)-3] + "." + hostParts[len(hostParts)-2] + "." + hostParts[len(hostParts)-1], nil
		}

		// 如果域名以 www 开头，特殊处理
		if hostParts[0] == "www" {
			return hostParts[len(hostParts)-2] + "." + hostParts[len(hostParts)-1], nil
		}

		return hostParts[len(hostParts)-2] + "." + hostParts[len(hostParts)-1], nil
	}
	return input, fmt.Errorf("输入既不是有效的 URL，也不是有效的 IP 地址")
}

// GetDomain 提取URL中的域名
func GetDomain(rawUrl string) (string, error) {
	// 解析 URL
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return rawUrl, nil
	}
	// 提取域名
	domain := parsedUrl.Host
	// 去掉端口号（如果有）
	domain = strings.Split(domain, ":")[0]
	return domain, nil
}

var (
	// 浏览器类型
	browsers = []string{
		"Chrome",
		"Firefox",
		"Safari",
		"Edge",
	}

	// 平台类型
	platforms = []string{
		"Windows NT 10.0; Win64; x64",
		"Windows NT 10.0; WOW64",
		"Windows NT 10.0",
		"Macintosh; Intel Mac OS X 10_15_7",
		"Macintosh; Intel Mac OS X 10_14_6",
		"X11; Ubuntu; Linux x86_64",
	}

	// 随机数生成器
	randSource = rand.New(rand.NewSource(time.Now().UnixNano()))
	// 互斥锁
	randMutex = sync.Mutex{}
)

// RandomUA 生成随机ua头
func RandomUA() string {
	// 加锁保证并发安全
	randMutex.Lock()
	defer randMutex.Unlock()

	// 预计算长度避免重复调用
	browser := browsers[randSource.Intn(len(browsers))]
	platform := platforms[randSource.Intn(len(platforms))]

	// 生成随机的版本号
	majorVersion := randSource.Intn(90) + 10
	buildVersion := randSource.Intn(4000) + 1000
	patchVersion := randSource.Intn(100) + 1
	version := fmt.Sprintf("%d.0.%d.%d", majorVersion, buildVersion, patchVersion)

	// 根据浏览器生成 User-Agent
	var userAgent string
	switch browser {
	case "Chrome":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", platform, version)
	case "Firefox":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s; rv:%d.0) Gecko/20100101 Firefox/%s", platform, randSource.Intn(90)+10, version)
	case "Safari":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%d.0 Safari/605.1.15", platform, randSource.Intn(14)+10)
	case "Edge":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s", platform, version, version)
	}

	return userAgent
}
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// HexDecode 16进制解码
func HexDecode(s string) []byte {
	dst := make([]byte, hex.DecodedLen(len(s))) //申请一个切片, 指明大小. 必须使用hex.DecodedLen
	n, err := hex.Decode(dst, []byte(s))        //进制转换, src->dst
	if err != nil {
		return nil
	}
	return dst[:n] //返回0:n的数据.
}

// FromHex 将十六进制字符串转换为普通字符串
func FromHex(data string) string {
	newStr, err := hex.DecodeString(data)
	if err == nil {
		return string(newStr)
	}
	return data
}

// HexEncode 字符串转为16进制
func HexEncode(s string) []byte {
	dst := make([]byte, hex.EncodedLen(len(s))) //申请一个切片, 指明大小. 必须使用hex.EncodedLen
	n := hex.Encode(dst, []byte(s))             //字节流转化成16进制
	return dst[:n]
}

// Str2UTF8 字符串转 utf 8
func Str2UTF8(str string) string {
	if len(str) == 0 {
		return ""
	}
	if !utf8.ValidString(str) {
		return mahonia.NewDecoder("gb18030").ConvertString(str)
	}
	return str
}

// Mmh3Hash32 计算 mmh3 hash
func Mmh3Hash32(raw []byte) int32 {
	var h32 = murmur3.New32()
	_, err := h32.Write(raw)
	if err != nil {
		return 0
	}
	return int32(h32.Sum32())
}

// Base64Encode base64 encode
func Base64Encode(braw []byte) []byte {
	bckd := base64.StdEncoding.EncodeToString(braw)
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

// RandLetters 随机小写字母
func RandLetters(n int) string {
	return RandFromChoices(n, letterBytes)
}

// RandFromChoices 从choices里面随机获取
func RandFromChoices(n int, choices string) string {
	b := make([]byte, n)
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, randSource.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSource.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(choices) {
			b[i] = choices[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// RandomString 生成随机字符串
func RandomString(len int) string {
	var container string
	var str = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := bytes.NewBufferString(str)
	length := b.Len()
	bigInt := big.NewInt(int64(length))
	for i := 0; i < len; i++ {
		randomInt, _ := rand2.Int(rand2.Reader, bigInt)
		container += string(str[randomInt.Int64()])
	}
	return container
}

// DealMultipart 处理multipart的/n问题
func DealMultipart(contentType string, ruleBody string) (result string, err error) {
	re := regexp.MustCompile(`(?m)multipart/form-Data; boundary=(.*)`)
	match := re.FindStringSubmatch(contentType)
	if len(match) != 2 {
		return "", errors.New("no boundary in content-type")
	}
	boundary := "--" + match[1]

	// 处理rule
	multiPartContent := ""
	multiFile := strings.Split(ruleBody, boundary)
	if len(multiFile) == 0 {
		return multiPartContent, errors.New("ruleBody.Body multi content format err")
	}

	for _, singleFile := range multiFile {
		//	文件头和文件响应
		SplitTmp := strings.Split(singleFile, "\n\n")
		if len(SplitTmp) == 2 {
			fileHeader := SplitTmp[0]
			fileBody := SplitTmp[1]
			fileHeader = strings.Replace(fileHeader, "\n", "\r\n", -1)
			multiPartContent += boundary + fileHeader + "\r\n\r\n" + strings.TrimRight(fileBody, "\n") + "\r\n"
		}
	}
	multiPartContent += boundary + "--" + "\r\n"
	return multiPartContent, nil
}

// ParseTarget 处理请求url地址
func ParseTarget(target, path string) string {
	// 去除末尾的斜杠
	target = strings.TrimRight(target, "/")
	if path == "" {
		return target
	}
	// 否则，直接将整个 path 附加到 target 后面
	return target + path
}

func UrlTypeToString(u *proto.UrlType) string {
	var buf strings.Builder
	if u.Scheme != "" {
		buf.WriteString(u.Scheme)
		buf.WriteByte(':')
	}
	if u.Scheme != "" || u.Host != "" {
		if u.Host != "" || u.Path != "" {
			buf.WriteString("//")
		}
		if h := u.Host; h != "" {
			buf.WriteString(u.Host)
		}
	}
	path := u.Path
	if path != "" && path[0] != '/' && u.Host != "" {
		buf.WriteByte('/')
	}
	if buf.Len() == 0 {
		if i := strings.IndexByte(path, ':'); i > -1 && strings.IndexByte(path[:i], '/') == -1 {
			buf.WriteString("./")
		}
	}
	buf.WriteString(path)

	if u.Query != "" {
		buf.WriteByte('?')
		buf.WriteString(u.Query)
	}
	if u.Fragment != "" {
		buf.WriteByte('#')
		buf.WriteString(u.Fragment)
	}
	return buf.String()
}

func Url2UrlType(u *url.URL) *proto.UrlType {
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

func ParseUrl(u *url.URL) *proto.UrlType {
	nu := &proto.UrlType{}
	nu.Scheme = u.Scheme
	nu.Domain = u.Hostname()
	nu.Host = u.Host
	nu.Port = u.Port()
	nu.Path = u.EscapedPath()
	nu.Query = u.RawQuery
	nu.Fragment = u.Fragment
	return nu
}

// ParseAddress 解析地址并返回 AddressInfo
func ParseAddress(address string) (AddressInfo, error) {
	var info AddressInfo

	// 检查是否以 https 或 http 开头
	if strings.HasPrefix(address, "https://") {
		info.Scheme = "https"
		info.IsLts = true // 如果是 https 开头，直接设置 LTS 为 true
	} else if strings.HasPrefix(address, "http://") {
		info.Scheme = "http"
	} else {
		// 如果没有指定协议，默认使用 https
		info.Scheme = "https"
	}

	// 如果地址没有协议部分，补全协议以便解析
	fullURL := address
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		fullURL = fmt.Sprintf("%s://%s", info.Scheme, address)
	}

	// 使用 url.Parse 解析地址
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return info, err
	}

	// 提取主机和端口
	host, port, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		// 如果没有显式的端口，使用默认端口
		host = parsedURL.Host
		if info.Scheme == "https" {
			port = "443"
			info.IsLts = true // https 默认端口 443，设置 LTS 为 true
		} else if info.Scheme == "http" {
			port = "80"
		}
	} else {
		// 如果显式指定了端口，检查是否是默认的 TLS 端口
		if info.Scheme == "https" && port == "443" {
			info.IsLts = true
		}
	}

	info.Host = host
	info.Port = port
	info.Hostname = parsedURL.Hostname()

	return info, nil
}

// GetRandomIP 获取随机ip地址
func GetRandomIP() string {
	//rand.Seed(time.Now().UnixNano())
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, randSource.Uint32())
	return ip.String()
}

// RemoveDuplicateURLs 去除重复的URL
func RemoveDuplicateURLs(urls []string) []string {
	// 使用map来判断URL是否重复
	urlMap := make(map[string]bool)
	var result []string

	for _, u := range urls {
		// 规范化URL，移除前后空格
		u = strings.TrimSpace(u)
		// 跳过空URL
		if u == "" {
			continue
		}

		// 如果URL不在map中，添加到结果中
		if !urlMap[u] {
			urlMap[u] = true
			result = append(result, u)
		}
	}

	return result
}

// URLDecode 解码URL编码的字符串
func URLDecode(encodedString string) (string, error) {
	// 替换+为空格
	encodedString = strings.Replace(encodedString, "+", " ", -1)

	// 解码URL编码的字符串
	decodedString, err := url.QueryUnescape(encodedString)
	if err != nil {
		return encodedString, err
	}

	return decodedString, nil
}

// URLEncode 编码字符串为URL编码格式
func URLEncode(rawString string) string {
	// 使用QueryEscape进行URL编码
	encodedString := url.QueryEscape(rawString)

	// 根据RFC 3986，某些字符在URL的路径部分不需要编码
	encodedString = strings.Replace(encodedString, "+", "%20", -1)

	return encodedString
}

// MD5Hash 生成字符串的 MD5 哈希值
func MD5Hash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// SHA1Hash 生成字符串的 SHA-1 哈希值
func SHA1Hash(input string) string {
	hash := sha1.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// SHA256Hash 生成字符串的 SHA-256 哈希值
func SHA256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// SHA512Hash 生成字符串的 SHA-512 哈希值
func SHA512Hash(input string) string {
	hash := sha512.Sum512([]byte(input))
	return hex.EncodeToString(hash[:])
}

// RemoveTrailingSlash 删除URL中的最后一个/
func RemoveTrailingSlash(url string) string {
	if strings.HasSuffix(url, "/") {
		return strings.TrimSuffix(url, "/")
	}
	return url
}
