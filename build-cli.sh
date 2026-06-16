#!/usr/bin/env bash
# 编译命令行版本 mlg-cli（纯 Go，无需 CGO / GCC / OpenGL）
# for Linux / macOS
set -e

CGO_ENABLED=0 go build -ldflags="-s -w" -o mlg-cli ./cmd/mlg-cli

echo "Build succeeded: ./mlg-cli"
