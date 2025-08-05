package headless_browser

import (
	"testing"
	"time"
)

func TestNavigate(t *testing.T) {

	b := NewBrowser(WithHeadless(false))
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	page.
		Timeout(30 * time.Second).
		MustNavigate("https://www.haha.ai").
		MustWaitStable()

	time.Sleep(1 * time.Second)

}
