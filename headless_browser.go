package headless_browser

import (
	"encoding/json"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
)

type Browser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
}

type Config struct {
	Headless  bool
	UserAgent string
	Cookies   string
}

type Option func(*Config)

func newDefaultConfig() *Config {
	return &Config{
		Headless:  true,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		Cookies:   "",
	}
}

func WithHeadless(headless bool) Option {
	return func(c *Config) {
		c.Headless = headless
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *Config) {
		c.UserAgent = userAgent
	}
}

func WithCookies(cookies string) Option {

	return func(c *Config) {
		c.Cookies = cookies
	}
}

func New(options ...Option) *Browser {

	cfg := newDefaultConfig()
	for _, option := range options {
		option(cfg)
	}

	launcher := launcher.New().
		Headless(cfg.Headless).
		Set("--no-sandbox").
		Set(
			"user-agent", cfg.UserAgent,
		)

	url := launcher.MustLaunch()

	browser := rod.New().
		ControlURL(url).
		Trace(true).
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
		browser:  browser,
		launcher: launcher,
	}
}

func (b *Browser) Close() {
	b.browser.MustClose()
	b.launcher.Cleanup()
}

func (b *Browser) NewPage() *rod.Page {
	return stealth.MustPage(b.browser)
}
