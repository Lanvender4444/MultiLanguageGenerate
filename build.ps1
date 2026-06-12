$env:PATH = "C:\winlibs\mingw64\bin;" + $env:PATH
# 代理设置，国内用户可以使用 https://goproxy.cn 以加速依赖下载
# $env:GOPROXY = "https://goproxy.cn,direct"
$env:CGO_ENABLED = "1"

# 检查并生成 Windows 资源文件（将 ico 图标编译为 app.syso），若不需要图标可跳过此步骤
if (Test-Path "public/icon.ico") {
    Write-Host "Generating Windows resource with icon..." -ForegroundColor Cyan
    rsrc -ico public/icon.ico -o app.syso
} else {
    Write-Host "Warning: public/icon.ico not found, historical icon might not be applied." -ForegroundColor Yellow
}

# 执行打包
go build -ldflags="-H=windowsgui -s -w" -o MultiLanguageGenerate.exe .

# 清理生成的临时资源文件（可选，保持目录干净）
if (Test-Path "app.syso") {
    Remove-Item "app.syso"
}

# 检查结果
if ($LASTEXITCODE -eq 0) {
    Write-Host "Build succeeded: MultiLanguageGenerate.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}