# Chrome Path Example

测试如何指定自定义 Chrome 路径的示例程序。

## 使用方法

```bash
# 构建
go build -o chrome_path_example

# 显示当前系统的常见 Chrome 路径
./chrome_path_example -show-paths

# 自动检测系统中的 Chrome
./chrome_path_example -auto-detect

# 使用自定义路径 (macOS 示例)
./chrome_path_example -chrome-path="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

# 使用自定义路径并在非 headless 模式运行（可以看到浏览器窗口）
./chrome_path_example -chrome-path="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" -headless=false

# 让 launcher 自动下载或查找 Chrome（不指定任何路径）
./chrome_path_example
```

## 命令行参数

- `-chrome-path string`: 自定义 Chrome/Chromium 可执行文件路径
- `-headless`: 在 headless 模式下运行（默认: false，会显示浏览器窗口）
- `-auto-detect`: 使用 launcher.LookPath() 自动检测 Chrome 路径
- `-show-paths`: 显示当前操作系统的常见 Chrome 路径

## 测试功能

程序会：
1. 根据参数创建浏览器实例
2. 导航到 example.com
3. 获取并打印页面标题
4. 验证浏览器功能正常