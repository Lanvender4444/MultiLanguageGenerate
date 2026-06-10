# MultiLanguageGenerate
利用 Golang + LLM API， 批量生成翻译文件。

## 自己编译
### Windows
1. 安装 GCC（仅首次需要）
从 https://github.com/brechtsanders/winlibs_mingw/releases 下载 winlibs-x86_64-posix-seh-gcc-*-ucrt-*.zip，解压到 C:\winlibs\，确保 C:\winlibs\mingw64\bin\gcc.exe 存在。

还需从 zip 中提取 mingw64/x86_64-w64-mingw32/ 目录到 C:\winlibs\mingw64\x86_64-w64-mingw32\。

2. 编译
```powershell
$env:PATH = "C:\winlibs\mingw64\bin;" + $env:PATH
$env:GOPROXY = "https://goproxy.cn,direct"
$env:CGO_ENABLED = "1"
go build -ldflags="-s -w" -o MultiLanguageGenerate.exe .
```

