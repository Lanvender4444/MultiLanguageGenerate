$env:PATH = "C:\winlibs\mingw64\bin;" + $env:PATH

# 如果在国内则代理
# $env:GOPROXY = "https://goproxy.cn,direct"
$env:CGO_ENABLED = "1"
go run .