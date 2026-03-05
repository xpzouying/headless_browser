package headless_browser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

// ProxyForwarder 本地代理转发器，用于支持带认证的远程代理
type ProxyForwarder struct {
	localAddr   string       // 本地监听地址
	remoteProxy *url.URL     // 远程代理地址
	server      *http.Server // HTTP 服务器
	listener    net.Listener // 监听器
	running     bool         // 是否正在运行
	mu          sync.Mutex   // 并发锁
	connectOnce sync.Once    // 确保只启动一次
}

// NewProxyForwarder 创建代理转发器
// proxyURL 格式: "http://user:pass@host:port" 或 "socks5://user:pass@host:port"
func NewProxyForwarder(proxyURL string) (*ProxyForwarder, error) {
	// 解析代理 URL
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	// 获取一个可用的本地端口
	addr, err := getAvailableLocalAddr()
	if err != nil {
		return nil, fmt.Errorf("failed to get local address: %w", err)
	}

	return &ProxyForwarder{
		localAddr:   addr,
		remoteProxy: parsed,
	}, nil
}

// Start 启动本地代理转发服务器
func (pf *ProxyForwarder) Start() error {
	var startErr error
	pf.connectOnce.Do(func() {
		pf.mu.Lock()
		defer pf.mu.Unlock()

		// 创建监听器
		listener, err := net.Listen("tcp", pf.localAddr)
		if err != nil {
			startErr = fmt.Errorf("failed to listen on %s: %w", pf.localAddr, err)
			return
		}
		pf.listener = listener

		// 获取实际监听地址（可能分配了不同端口）
		pf.localAddr = listener.Addr().String()

		// 创建 HTTP 代理服务器
		pf.server = &http.Server{
			Handler: http.HandlerFunc(pf.handleRequest),
		}

		pf.running = true

		// 启动服务器（异步）
		go func() {
			if err := pf.server.Serve(listener); err != nil && err != http.ErrServerClosed {
				logrus.Errorf("proxy forwarder error: %v", err)
			}
		}()

		logrus.Infof("Proxy forwarder started: %s -> %s",
			pf.localAddr, maskURL(pf.remoteProxy))
	})

	return startErr
}

// GetLocalAddr 获取本地代理地址
func (pf *ProxyForwarder) GetLocalAddr() string {
	return pf.localAddr
}

// Stop 停止代理转发器
func (pf *ProxyForwarder) Stop() error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if !pf.running {
		return nil
	}

	pf.running = false

	if pf.server != nil {
		if err := pf.server.Shutdown(context.Background()); err != nil {
			// Shutdown 失败可能是正常的，继续关闭
			logrus.Debugf("server shutdown error: %v", err)
		}
	}

	if pf.listener != nil {
		if err := pf.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}

	logrus.Info("Proxy forwarder stopped")
	return nil
}

// handleRequest 处理代理请求
func (pf *ProxyForwarder) handleRequest(w http.ResponseWriter, r *http.Request) {
	// 确保请求 URI 为空（代理模式）
	r.RequestURI = ""

	// 根据代理类型处理
	if pf.remoteProxy.Scheme == "socks5" {
		pf.handleViaSOCKS5(w, r)
	} else {
		pf.handleViaHTTP(w, r)
	}
}

// handleViaHTTP 通过 HTTP/HTTPS 代理转发请求
func (pf *ProxyForwarder) handleViaHTTP(w http.ResponseWriter, r *http.Request) {
	// 创建 HTTP 传输层，通过远程代理
	transport := &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			return pf.remoteProxy, nil
		},
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Transport: transport,
	}

	// 发送请求
	resp, err := client.Do(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
		logrus.Errorf("HTTP proxy error: %v", err)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 设置状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	if _, err := io.Copy(w, resp.Body); err != nil {
		logrus.Errorf("error copying response: %v", err)
	}
}

// handleViaSOCKS5 通过 SOCKS5 代理转发请求
func (pf *ProxyForwarder) handleViaSOCKS5(w http.ResponseWriter, r *http.Request) {
	// 处理 CONNECT 方法（用于 HTTPS）
	if r.Method == "CONNECT" {
		pf.handleHTTPSViaSOCKS5(w, r)
		return
	}

	// HTTP 请求
	pf.handleHTTPViaSOCKS5(w, r)
}

// handleHTTPViaSOCKS5 通过 SOCKS5 代理转发 HTTP 请求
func (pf *ProxyForwarder) handleHTTPViaSOCKS5(w http.ResponseWriter, r *http.Request) {
	// 创建 SOCKS5 拨号器
	dialer, err := pf.createSOCKS5Dialer()
	if err != nil {
		http.Error(w, fmt.Sprintf("socks5 error: %v", err), http.StatusBadGateway)
		return
	}

	// 构建目标地址
	targetHost := r.URL.Host
	if targetHost == "" {
		targetHost = r.Host
	}

	// 通过 SOCKS5 连接
	conn, err := dialer.(interface {
		Dial(network, addr string) (net.Conn, error)
	}).Dial("tcp", targetHost)
	if err != nil {
		http.Error(w, fmt.Sprintf("socks5 connect error: %v", err), http.StatusBadGateway)
		logrus.Errorf("SOCKS5 dial error: %v", err)
		return
	}
	defer conn.Close()

	// 发送 HTTP 请求
	if err := r.Write(conn); err != nil {
		http.Error(w, fmt.Sprintf("socks5 write error: %v", err), http.StatusBadGateway)
		return
	}

	// 读取响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), r)
	if err != nil {
		http.Error(w, fmt.Sprintf("socks5 read error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 设置状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	if _, err := io.Copy(w, resp.Body); err != nil {
		logrus.Errorf("error copying response: %v", err)
	}
}

// handleHTTPSViaSOCKS5 通过 SOCKS5 代理处理 HTTPS CONNECT 请求
func (pf *ProxyForwarder) handleHTTPSViaSOCKS5(w http.ResponseWriter, r *http.Request) {
	// 创建 SOCKS5 拨号器
	dialer, err := pf.createSOCKS5Dialer()
	if err != nil {
		http.Error(w, fmt.Sprintf("socks5 error: %v", err), http.StatusBadGateway)
		return
	}

	// 获取目标主机和端口
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}

	// 通过 SOCKS5 连接目标服务器
	conn, err := dialer.(interface {
		Dial(network, addr string) (net.Conn, error)
	}).Dial("tcp", host)
	if err != nil {
		http.Error(w, fmt.Sprintf("socks5 connect error: %v", err), http.StatusServiceUnavailable)
		logrus.Errorf("SOCKS5 CONNECT error: %v", err)
		return
	}

	// 发送 200 Connection established
	w.WriteHeader(http.StatusOK)

	// 获取底层的 TCP 连接（如果支持 hijack）
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		conn.Close()
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("hijack error: %v", err), http.StatusInternalServerError)
		conn.Close()
		return
	}
	defer clientConn.Close()
	defer conn.Close()

	// 双向转发数据
	go func() {
		defer clientConn.Close()
		defer conn.Close()
		io.Copy(conn, clientConn)
	}()
	io.Copy(clientConn, conn)
}

// createSOCKS5Dialer 创建 SOCKS5 拨号器
func (pf *ProxyForwarder) createSOCKS5Dialer() (proxy.Dialer, error) {
	// 从代理 URL 中获取认证信息
	var auth *proxy.Auth
	if pf.remoteProxy.User != nil {
		password, _ := pf.remoteProxy.User.Password()
		auth = &proxy.Auth{
			User:     pf.remoteProxy.User.Username(),
			Password: password,
		}
	}

	// 创建 SOCKS5 代理地址
	proxyAddr := fmt.Sprintf("%s:%s", pf.remoteProxy.Hostname(), pf.remoteProxy.Port())

	// 创建 SOCKS5 拨号器
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	return dialer, nil
}

// getAvailableLocalAddr 获取可用的本地地址
func getAvailableLocalAddr() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	return listener.Addr().String(), nil
}

// maskURL 隐藏 URL 中的敏感信息
func maskURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	masked := *u
	if masked.User != nil {
		masked.User = url.UserPassword("****", "****")
	}
	return masked.String()
}
