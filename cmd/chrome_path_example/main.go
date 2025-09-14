package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"

	"github.com/go-rod/rod/lib/launcher"
	headless_browser "github.com/xpzouying/headless_browser"
)

func main() {
	var (
		chromePath = flag.String("chrome-path", "", "Custom Chrome/Chromium executable path")
		headless   = flag.Bool("headless", false, "Run in headless mode")
		autoDetect = flag.Bool("auto-detect", false, "Auto-detect Chrome path using launcher.LookPath()")
		showPaths  = flag.Bool("show-paths", false, "Show common Chrome paths for current OS")
	)
	flag.Parse()

	if *showPaths {
		showCommonPaths()
		return
	}

	// 确定要使用的 Chrome 路径
	var finalPath string
	if *autoDetect {
		if path, exists := launcher.LookPath(); exists {
			finalPath = path
			fmt.Printf("Auto-detected Chrome path: %s\n", path)
		} else {
			log.Fatal("Could not auto-detect Chrome path")
		}
	} else if *chromePath != "" {
		finalPath = *chromePath
		fmt.Printf("Using custom Chrome path: %s\n", finalPath)
	} else {
		fmt.Println("No Chrome path specified, launcher will auto-download or find Chrome")
	}

	// 创建浏览器实例
	var browser *headless_browser.Browser
	if finalPath != "" {
		browser = headless_browser.New(
			headless_browser.WithChromeBinPath(finalPath),
			headless_browser.WithHeadless(*headless),
		)
	} else {
		browser = headless_browser.New(
			headless_browser.WithHeadless(*headless),
		)
	}
	defer browser.Close()

	// 测试导航
	fmt.Println("Creating new page...")
	page := browser.NewPage()

	fmt.Println("Navigating to example.com...")
	page.MustNavigate("https://example.com")

	title := page.MustInfo().Title
	fmt.Printf("Page title: %s\n", title)

	fmt.Println("Test completed successfully!")
}

func showCommonPaths() {
	fmt.Println("Common Chrome/Chromium paths for", runtime.GOOS)
	fmt.Println()

	switch runtime.GOOS {
	case "darwin":
		fmt.Println("macOS:")
		fmt.Println("  Chrome:")
		fmt.Println("    /Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
		fmt.Println("  Chrome Canary:")
		fmt.Println("    /Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary")
		fmt.Println("  Chromium:")
		fmt.Println("    /Applications/Chromium.app/Contents/MacOS/Chromium")
		fmt.Println("  Edge:")
		fmt.Println("    /Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge")
		fmt.Println("  Brave:")
		fmt.Println("    /Applications/Brave Browser.app/Contents/MacOS/Brave Browser")

	case "linux":
		fmt.Println("Linux:")
		fmt.Println("  Chrome:")
		fmt.Println("    /usr/bin/google-chrome")
		fmt.Println("    /opt/google/chrome/chrome")
		fmt.Println("  Chromium:")
		fmt.Println("    /usr/bin/chromium")
		fmt.Println("    /usr/bin/chromium-browser")
		fmt.Println("    /snap/bin/chromium")
		fmt.Println("  Edge:")
		fmt.Println("    /usr/bin/microsoft-edge")
		fmt.Println("    /opt/microsoft/msedge/msedge")

	case "windows":
		fmt.Println("Windows:")
		fmt.Println("  Chrome:")
		fmt.Println("    C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe")
		fmt.Println("    C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe")
		fmt.Println("  Edge:")
		fmt.Println("    C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe")
		fmt.Println("  Chromium:")
		fmt.Println("    C:\\Program Files\\Chromium\\Application\\chrome.exe")

	default:
		fmt.Println("Unknown OS:", runtime.GOOS)
	}

	fmt.Println("\nTip: Use -auto-detect flag to automatically find Chrome on your system")
}
