package network

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

const (
	DefaultNetwork      = "tcp"
	DefaultDialTimeout  = 5 * time.Second
	DefaultWriteTimeout = 5 * time.Second
	DefaultReadTimeout  = 5 * time.Second
	DefaultRetryDelay   = 2 * time.Second
	DefaultReadSize     = 2048
	DefaultMaxRetries   = 3
)

// TcpOrUdpConfig 配置结构体
type TcpOrUdpConfig struct {
	Network      string        // 网络类型，TCP 或 UDP
	MaxRetries   int           // 最大重试次数
	ReadSize     int           // 读取数据的缓冲区大小
	DialTimeout  time.Duration // 连接超时时间
	WriteTimeout time.Duration // 写入超时时间
	ReadTimeout  time.Duration // 读取超时时间
	RetryDelay   time.Duration // 重试延迟时间
	ProxyURL     string        // 代理URL
	IsLts        bool          // 是否发送LTS请求
	ServerName   string        // ServerName对tls请求的配置
}

// Client 客户端结构体
type Client struct {
	address string
	conn    net.Conn
	conf    TcpOrUdpConfig
}

// parseAddress 解析地址，确保包含端口号
func parseAddress(address string) string {
	// 检查是否已经包含端口号
	if _, _, err := net.SplitHostPort(address); err == nil {
		return address
	}

	// 检查是否是 HTTPS 地址
	if strings.HasPrefix(address, "https://") {
		// 移除 https:// 前缀
		address = strings.TrimPrefix(address, "https://")
		return net.JoinHostPort(address, "443")
	}

	// 检查是否是 HTTP 地址
	if strings.HasPrefix(address, "http://") {
		// 移除 http:// 前缀
		address = strings.TrimPrefix(address, "http://")
		return net.JoinHostPort(address, "80")
	}

	// 默认使用 80 端口
	return net.JoinHostPort(address, "80")
}

// NewClient 创建新客户端
func NewClient(address string, conf TcpOrUdpConfig) (*Client, error) {
	var (
		err  error
		conn net.Conn
	)

	// 解析地址，确保包含端口号
	address = parseAddress(address)

	// 设置默认值
	if conf.DialTimeout == 0 {
		conf.DialTimeout = DefaultDialTimeout
	}

	if conf.RetryDelay == 0 {
		conf.RetryDelay = DefaultRetryDelay
	}

	if conf.MaxRetries == 0 {
		conf.MaxRetries = DefaultMaxRetries
	}

	if len(conf.Network) == 0 {
		conf.Network = DefaultNetwork
	}

	// 创建Dialer
	var dialer proxy.Dialer = &net.Dialer{Timeout: conf.DialTimeout}

	// 处理代理
	if conf.ProxyURL != "" {
		proxyURL, err := url.Parse(conf.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		//dialer, err = proxyclient.NewClient(proxyURL)
		dialer, err = proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}
	}

	// 尝试连接
	for i := 0; i < conf.MaxRetries; i++ {
		conn, err = dialer.Dial(conf.Network, address)
		if err == nil {
			if conf.Network == "tcp" && conf.IsLts {
				// 使用TLS
				tlsConn := tls.Client(conn, &tls.Config{
					InsecureSkipVerify: true,
					ServerName:         conf.ServerName, // 动态配置 ServerName
				})
				err = tlsConn.Handshake()
				if err == nil {
					conn = tlsConn
					break
				} else {
					_ = conn.Close()
				}
			} else {
				break
			}
		}
		time.Sleep(conf.RetryDelay)
	}

	if err != nil {
		return nil, err
	}

	return &Client{address: address, conn: conn, conf: conf}, nil
}

// Send 发送数据
func (c *Client) Send(data []byte) error {
	if c.conn == nil {
		return errors.New("connection is not established")
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout()))
	_, err := c.conn.Write(data)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return c.retryWrite(data)
		}
		return err
	}
	return nil
}

// Receive 接收数据
func (c *Client) Receive() ([]byte, error) {
	if c.conn == nil {
		return nil, errors.New("connection is not established")
	}

	_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout()))
	buf := make([]byte, c.readSize())
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, c.retryRead(buf)
	}
	return buf[:n], nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// retryWrite 重试写入数据
func (c *Client) retryWrite(data []byte) error {
	for i := 0; i < c.maxRetries(); i++ {
		time.Sleep(c.retryTimeout())
		conn, err := net.DialTimeout(c.network(), c.conn.RemoteAddr().String(), c.dialTimeout())
		if err == nil {
			c.conn = conn
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout()))
			_, err = c.conn.Write(data)
			if err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("failed to send data after %d retries: %w", c.maxRetries(), errors.New("connection is closed"))
}

// retryRead 重试读取数据
func (c *Client) retryRead(buf []byte) error {
	for i := 0; i < c.maxRetries(); i++ {
		time.Sleep(c.retryTimeout())
		conn, err := net.DialTimeout(c.network(), c.conn.RemoteAddr().String(), c.dialTimeout())
		if err == nil {
			c.conn = conn
			_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout()))
			n, err := c.conn.Read(buf)
			if err == nil && n > 0 {
				return nil
			}
		}
	}
	return fmt.Errorf("failed to receive data after %d retries: %w", c.maxRetries(), errors.New("connection is closed"))
}

func (c *Client) network() string {
	if len(c.conf.Network) > 0 {
		return c.conf.Network
	}
	return DefaultNetwork
}

func (c *Client) dialTimeout() time.Duration {
	if c.conf.DialTimeout != 0 {
		return c.conf.DialTimeout
	}
	return DefaultDialTimeout
}

func (c *Client) writeTimeout() time.Duration {
	if c.conf.WriteTimeout != 0 {
		return c.conf.WriteTimeout
	}
	return DefaultWriteTimeout
}

func (c *Client) readTimeout() time.Duration {
	if c.conf.ReadTimeout != 0 {
		return c.conf.ReadTimeout
	}
	return DefaultReadTimeout
}

func (c *Client) retryTimeout() time.Duration {
	if c.conf.RetryDelay != 0 {
		return c.conf.RetryDelay
	}
	return DefaultRetryDelay
}

func (c *Client) maxRetries() int {
	if c.conf.MaxRetries != 0 {
		return c.conf.MaxRetries
	}
	return DefaultMaxRetries
}

func (c *Client) readSize() int {
	if c.conf.ReadSize != 0 {
		return c.conf.ReadSize
	}
	return DefaultReadSize
}

// NewTcpClient 创建新的TCP客户端
func NewTcpClient(address string, conf TcpOrUdpConfig) (*Client, error) {
	conf.Network = "tcp"
	address = parseAddress(address)
	return NewClient(address, conf)
}

func (c *Client) SendTcp(data []byte) error {
	if c.conf.IsLts {
		return c.SendLtsTcp(data)
	}
	return c.Send(data)
}

func (c *Client) RecvTcp() ([]byte, error) {
	if c.conf.IsLts {
		return c.RecvLtsTcp()
	}
	return c.Receive()
}

// NewUdpClient 创建新的UDP客户端
func NewUdpClient(address string, conf TcpOrUdpConfig) (*Client, error) {
	conf.Network = "udp"
	address = parseAddress(address)
	return NewClient(address, conf)
}

func (c *Client) SendUDP(data []byte) error {
	if c.conf.IsLts {
		return c.SendLtsUdp(data)
	}
	return c.Send(data)
}

func (c *Client) RecvUdp() ([]byte, error) {
	if c.conf.IsLts {
		return c.RecvLtsUdp()
	}
	return c.Receive()
}

// NewLtsTcpClient 创建新的LTS TCP客户端
func NewLtsTcpClient(address string, conf TcpOrUdpConfig) (*Client, error) {
	conf.Network = "tcp"
	conf.IsLts = true
	address = parseAddress(address)
	return NewClient(address, conf)
}

func (c *Client) SendLtsTcp(data []byte) error {
	return c.Send(data)
}

func (c *Client) RecvLtsTcp() ([]byte, error) {
	return c.Receive()
}

// NewLtsUdpClient 创建新的LTS UDP客户端
func NewLtsUdpClient(address string, conf TcpOrUdpConfig) (*Client, error) {
	conf.Network = "udp"
	conf.IsLts = true
	address = parseAddress(address)
	return NewClient(address, conf)
}

func (c *Client) SendLtsUdp(data []byte) error {
	return c.Send(data)
}

func (c *Client) RecvLtsUdp() ([]byte, error) {
	return c.Receive()
}
