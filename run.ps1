$env:PATH = "C:\winlibs\mingw64\bin;" + $env:PATH
$env:GOPROXY = "https://goproxy.cn,direct"
$env:CGO_ENABLED = "1"
go run .