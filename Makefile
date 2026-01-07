.PHONY: build run migrate clean

# 编译
build:
	go build -o license-server.exe ./cmd/main.go

# 运行
run:
	go run ./cmd/main.go

# 数据库迁移
migrate:
	go run ./cmd/main.go -migrate

# 清理
clean:
	rm -f license-server.exe

# 安装依赖
deps:
	go mod tidy

# 格式化代码
fmt:
	go fmt ./...
