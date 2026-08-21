# btyscan 构建
#   make build        : 用 go.mod(依赖环境能解析 chain33 模块)
#   make build-local  : 用 go.mod.local(replace 到本地 chain33 源码, 离线可用)
#   make test         : 运行单元测试(本地 chain33 replace 版本)
#
# 默认 CGO_ENABLED=0 静态编译, 产物不依赖目标机 glibc 版本(可在老系统上直接运行)。
# 构建缓存默认放在 ./gocache(避免 ~/.cache/go-build 权限问题), 可用 GOCACHE=... 覆盖。
# build-local/test 通过 bash trap 保证无论构建成功/失败, go.mod 都会被恢复。

GO      ?= go
BIN     := btyscan
GOCACHE ?= $(CURDIR)/gocache
CGO     ?= 0

.PHONY: build build-local test clean

build:
	mkdir -p $(GOCACHE)
	CGO_ENABLED=$(CGO) GOCACHE=$(GOCACHE) $(GO) build -o $(BIN) .

build-local:
	@bash -c 'set -e; test -f go.mod.local || { echo "go.mod.local 不存在"; exit 1; }; \
		cp go.mod go.mod.clean; cp go.mod.local go.mod; \
		trap "mv go.mod.clean go.mod" EXIT; \
		mkdir -p $(GOCACHE); \
		CGO_ENABLED=$(CGO) GOFLAGS=-mod=mod GOPROXY=off GOSUMDB=off GOCACHE=$(GOCACHE) $(GO) build -o $(BIN) .'

test:
	@bash -c 'set -e; test -f go.mod.local || { echo "go.mod.local 不存在"; exit 1; }; \
		cp go.mod go.mod.clean; cp go.mod.local go.mod; \
		trap "mv go.mod.clean go.mod" EXIT; \
		mkdir -p $(GOCACHE); \
		CGO_ENABLED=$(CGO) GOFLAGS=-mod=mod GOPROXY=off GOSUMDB=off GOCACHE=$(GOCACHE) $(GO) test -count=1 .'

clean:
	rm -f $(BIN)
