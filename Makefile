# 顶层统一构建入口
# 为什么用递归 make -j：Go API / Next.js web / Vite acg-checker 三者互不依赖，
# 默认目标自动派发并行子构建，用户无需手动记 -j 参数。

SHELL := /bin/bash
ROOT  := $(shell pwd)
BIN   := $(ROOT)/bin

# Go API 的三个命令入口（cmd/ 下）
GO_CMDS := server seed sync

# Go 交叉编译参数：默认编本机；CGO 关闭以产出纯静态二进制（部署 Linux 无依赖）
GOOS        ?=
GOARCH      ?=
CGO_ENABLED ?= 0
# 本机产物进 bin/；交叉编译进 bin/<goos>_<goarch>/，与本机版互不覆盖
API_OUT := $(if $(GOOS),$(BIN)/$(GOOS)_$(GOARCH),$(BIN))

.PHONY: build build-all build-api build-linux build-web build-acg install clean help

# 默认目标：并行构建全部（递归调用自动加 -j3，无需手动指定）
build:
	@$(MAKE) -j3 build-all

build-all: build-api build-web build-acg

# Go API：编译 cmd 下三个入口，任一失败即中断
# GOOS/GOARCH 为空时编本机；可显式覆盖，如 make build-api GOOS=linux GOARCH=arm64
build-api:
	@echo "==> 构建 Go API (GOOS=$(if $(GOOS),$(GOOS),native) GOARCH=$(if $(GOARCH),$(GOARCH),native))"
	@cd apps/api && for c in $(GO_CMDS); do \
		echo "    go build cmd/$$c -> $(API_OUT)/$$c"; \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
			go build -o $(API_OUT)/$$c ./cmd/$$c || exit 1; \
	done

# 交叉编译 Linux 版（部署服务器用），产物在 bin/linux_amd64/
build-linux:
	@$(MAKE) build-api GOOS=linux GOARCH=amd64

# Next.js web（pnpm）
build-web:
	@echo "==> 构建 web (Next.js)"
	@cd apps/web && pnpm build

# Vite + React acg-checker（npm）
build-acg:
	@echo "==> 构建 acg-checker (Vite)"
	@cd apps/acg-checker && npm run build

# 一次性安装各 app 依赖
install:
	@cd apps/web && pnpm install
	@cd apps/acg-checker && npm install
	@cd apps/api && go mod download

# 清理构建产物
clean:
	@rm -rf $(BIN) apps/web/.next apps/acg-checker/dist

help:
	@echo "可用目标："
	@echo "  make build       并行构建全部 (api + web + acg-checker)"
	@echo "  make build-api   构建 Go API 到 bin/（本机；可传 GOOS/GOARCH 交叉编译）"
	@echo "  make build-linux 交叉编译 Linux 版 Go API 到 bin/linux_amd64/"
	@echo "  make build-web   仅构建 web"
	@echo "  make build-acg   仅构建 acg-checker"
	@echo "  make install     安装各 app 依赖"
	@echo "  make clean       清理构建产物"
