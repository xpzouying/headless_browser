# headless_browser

A simple headless browser library powered by go-rod, with built-in stealth mode support.

## Installation

```bash
go get github.com/xpzouying/headless_browser
```

## Usage

```go
package main

import (
    "time"
    
    "github.com/xpzouying/headless_browser"
)

func main() {
    // Create browser with default settings (headless mode)
    browser := headless_browser.New()
    defer browser.Close()
    
    // Create a new page
    page := browser.NewPage()
    defer page.Close()
    
    // Navigate to a website
    page.Timeout(30 * time.Second).
        MustNavigate("https://example.com").
        MustWaitStable()
}
```

## Configuration Options

```go
// Run in non-headless mode (visible browser)
browser := headless_browser.New(
    headless_browser.WithHeadless(false),
)

// Set custom user agent
browser := headless_browser.New(
    headless_browser.WithUserAgent("Custom User Agent"),
)

// Set cookies (JSON format)
browser := headless_browser.New(
    headless_browser.WithCookies(`[{"name":"session","value":"abc123","domain":"example.com"}]`),
)

// Combine multiple options
browser := headless_browser.New(
    headless_browser.WithHeadless(false),
    headless_browser.WithUserAgent("Custom User Agent"),
    headless_browser.WithCookies(cookiesJSON),
)
```

## Example

```go
package main

import (
    "time"
    
    "github.com/xpzouying/headless_browser"
)

func main() {
    // Create browser instance
    browser := headless_browser.New(headless_browser.WithHeadless(false))
    defer browser.Close()

    // Create new page with stealth mode
    page := browser.NewPage()
    defer page.Close()

    // Navigate and wait for page to be stable
    page.Timeout(30 * time.Second).
        MustNavigate("https://www.haha.ai").
        MustWaitStable()

    // Additional page operations can be performed here
    time.Sleep(1 * time.Second)
}
```
