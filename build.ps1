$env:PATH = "C:\winlibs\mingw64\bin;" + $env:PATH
$env:GOPROXY = "https://goproxy.cn,direct"
$env:CGO_ENABLED = "1"

go build -ldflags="-H=windowsgui -s -w" -o MultiLanguageGenerate.exe .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build succeeded: MultiLanguageGenerate.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}