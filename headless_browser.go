// Package headless_browser provides a simple headless browser library powered by go-rod with built-in stealth mode support.
package headless_browser

import (
	"encoding/json"
	"net/url"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
)

// Browser represents a headless browser instance with an underlying rod.Browser and launcher.
type Browser struct {
	browser         *rod.Browser
	launcher        *launcher.Launcher
	proxyForwarder  *ProxyForwarder // 代理转发器
}

// Config holds the configuration options for the browser.
type Config struct {
	Headless      bool   // Whether to run browser in headless mode
	UserAgent     string // Custom user agent string
	Cookies       string // JSON string of cookies to set
	ChromeBinPath string // Custom Chrome/Chromium executable path
	Proxy         string // Proxy server URL (e.g. "http://host:port", "socks5://user:pass@host:port")

	Trace bool // Whether to enable tracing (not implemented yet)
}

// Option is a functional option for configuring the browser.
type Option func(*Config)

// newDefaultConfig returns a new Config with default values.
func newDefaultConfig() *Config {
	return &Config{
		Headless:      true,
		UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		Cookies:       "",
		ChromeBinPath: "", // Empty means auto-detect
		Trace:         false,
	}
}

// WithHeadless sets whether the browser should run in headless mode.
func WithHeadless(headless bool) Option {
	return func(c *Config) {
		c.Headless = headless
	}
}

// WithUserAgent sets a custom user agent string for the browser.
func WithUserAgent(userAgent string) Option {
	return func(c *Config) {
		c.UserAgent = userAgent
	}
}

// WithCookies sets cookies for the browser from a JSON string.
// The cookies should be in the format expected by proto.NetworkCookie.
func WithCookies(cookies string) Option {
	return func(c *Config) {
		c.Cookies = cookies
	}
}

// WithChromeBinPath sets a custom Chrome/Chromium executable path.
// If not set or empty, launcher will auto-detect or download a browser.
// Common paths:
//   - macOS: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
//   - Linux: "/usr/bin/google-chrome" or "/usr/bin/chromium"
//   - Windows: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
func WithChromeBinPath(path string) Option {
	return func(c *Config) {
		c.ChromeBinPath = path
	}
}

// WithProxy sets a proxy server for all browser requests.
// Supports HTTP, HTTPS, and SOCKS5 proxies.
//
// Examples:
//   - Without authentication: "http://proxy.example.com:8080", "socks5://127.0.0.1:1080"
//   - With authentication: "http://user:pass@proxy.example.com:8080", "socks5://user:pass@127.0.0.1:1080"
//
// For authenticated proxies, a local forwarder will be automatically created
// to handle the authentication.
func WithProxy(proxy string) Option {
	return func(c *Config) {
		c.Proxy = proxy
	}
}

func WithTrace() Option {
	return func(c *Config) {
		c.Trace = true
	}
}

// New creates a new Browser instance with the provided options.
// It initializes a Chrome browser with stealth mode enabled.
func New(options ...Option) *Browser {
	cfg := newDefaultConfig()
	for _, option := range options {
		option(cfg)
	}

	l := launcher.New().
		Headless(cfg.Headless).
		Set("--no-sandbox").
		Set(
			"user-agent", cfg.UserAgent,
		)

	// Set custom Chrome binary path if provided
	if cfg.ChromeBinPath != "" {
		l = l.Bin(cfg.ChromeBinPath)
	}

	// 处理代理设置
	var proxyForwarder *ProxyForwarder
	proxyURL := cfg.Proxy

	if proxyURL != "" {
		// 检查是否需要代理转发（带认证的代理）
		parsed, err := url.Parse(proxyURL)
		if err == nil && parsed.User != nil {
			// 带认证的代理，创建本地转发器
			proxyForwarder, err = NewProxyForwarder(proxyURL)
			if err != nil {
				logrus.Errorf("failed to create proxy forwarder: %v", err)
				// 继续使用原代理 URL，让 Chrome 自己处理
			} else {
				if err := proxyForwarder.Start(); err != nil {
					logrus.Errorf("failed to start proxy forwarder: %v", err)
				} else {
					// 使用本地转发器地址
					proxyURL = "http://" + proxyForwarder.GetLocalAddr()
					logrus.Infof("Using authenticated proxy via local forwarder: %s", proxyURL)
				}
			}
		}

		// 设置代理
		l = l.Proxy(proxyURL)
	}

	url := l.MustLaunch()

	browser := rod.New().
		ControlURL(url).
		Trace(cfg.Trace).
		MustConnect()

	// 加载 cookies
	if cfg.Cookies != "" {
		var cookies []*proto.NetworkCookie
		if err := json.Unmarshal([]byte(cfg.Cookies), &cookies); err != nil {
			logrus.Warnf("failed to unmarshal cookies: %v", err)
		} else {
			browser.MustSetCookies(cookies...)
		}
	}

	return &Browser{
		browser:        browser,
		launcher:       l,
		proxyForwarder: proxyForwarder,
	}
}

// Close closes the browser and cleans up resources.
func (b *Browser) Close() {
	b.browser.MustClose()
	b.launcher.Cleanup()

	// 停止代理转发器
	if b.proxyForwarder != nil {
		if err := b.proxyForwarder.Stop(); err != nil {
			logrus.Errorf("failed to stop proxy forwarder: %v", err)
		}
	}
}

// NewPage creates a new page with stealth mode enabled.
// The returned page can be used to navigate and interact with web content.
func (b *Browser) NewPage() *rod.Page {
	return stealth.MustPage(b.browser)
}
