//go:build !windows

package headless_browser

import (
	"syscall"
	"testing"
	"time"
)

// 浏览器无响应时，Close 应超时兜底回收进程，而不是一直阻塞。
func TestCloseKillsUnresponsiveBrowser(t *testing.T) {
	b := New(WithHeadless(true))

	pid := b.launcher.PID()
	if pid == 0 {
		t.Fatal("拿不到浏览器进程 pid")
	}

	// 冻住进程，模拟 CDP 无响应
	if err := syscall.Kill(pid, syscall.SIGSTOP); err != nil {
		t.Fatalf("SIGSTOP failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		b.Close()
	}()

	select {
	case <-done:
	case <-time.After(closeTimeout + 2*cleanupTimeout + 10*time.Second):
		t.Fatal("Close 未在超时内返回")
	}

	time.Sleep(2 * time.Second)
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("进程 %d 未被回收", pid)
	}
}
