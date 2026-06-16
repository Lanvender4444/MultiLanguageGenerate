# 编译命令行版本 mlg-cli（纯 Go，无需 CGO / GCC / OpenGL）
# for Windows

$env:CGO_ENABLED = "0"

go build -ldflags="-s -w" -o mlg-cli.exe ./cmd/mlg-cli

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build succeeded: mlg-cli.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
